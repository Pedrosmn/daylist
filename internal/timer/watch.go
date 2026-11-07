package timer

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const defaultWatchPoll = time.Second

// watchExecutable é substituível nos testes para não iniciar o binário real.
var watchExecutable = os.Executable

// WatchOptions configura o daemon que observa o estado do timer e notifica o
// mínimo e o máximo. Os campos injetáveis existem para os testes.
type WatchOptions struct {
	StatePath string
	PIDPath   string
	Notify    func(title, body string) error
	Now       func() time.Time
	Sleep     func(time.Duration)
	Poll      time.Duration
}

// ThresholdUpdate é o resultado de uma verificação atômica dos limites.
type ThresholdUpdate struct {
	State   *State
	Message string
	Changed bool
}

// ClaimThresholds carrega o estado, verifica os limites no instante informado,
// persiste os sinalizadores e devolve a mensagem a notificar. A verificação é
// feita sob um lock de arquivo para que TUI, CLI e watch não notifiquem o mesmo
// limite duas vezes. Timer pausado não dispara notificação.
func ClaimThresholds(statePath string, now time.Time) (ThresholdUpdate, error) {
	release, err := lockFile(statePath + ".lock")
	if err != nil {
		return ThresholdUpdate{}, err
	}
	defer release()

	s, err := Load(statePath)
	if err != nil {
		return ThresholdUpdate{}, err
	}
	if s.Paused {
		return ThresholdUpdate{State: s}, nil
	}

	message, changed := s.CheckThresholds(s.TotalMsAt(now.UnixMilli()))
	if changed {
		if err := s.saveLocked(statePath); err != nil {
			return ThresholdUpdate{State: s}, err
		}
	}
	return ThresholdUpdate{State: s, Message: message, Changed: changed}, nil
}

// ClearState remove o estado do timer e encerra o watch, mantendo a remoção e
// a parada sob o mesmo lock usado pelas notificações.
func ClearState(statePath, pidPath string) error {
	if statePath == "" {
		return nil
	}
	release, err := lockFile(statePath + ".lock")
	if err != nil {
		return err
	}
	removeErr := os.Remove(statePath)
	release()
	if pidPath != "" {
		StopWatch(pidPath)
	}
	if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
		return removeErr
	}
	return nil
}

// Watch observa o timer até não haver mais limite a notificar. Ele sai quando o
// estado não existe, quando o timer não tem faixa de tempo, quando o máximo já
// foi notificado ou quando o lock indica que já existe outro watch rodando.
func Watch(opts WatchOptions) error {
	if opts.StatePath == "" {
		return errors.New("watch do timer: caminho do estado vazio")
	}
	if opts.PIDPath == "" {
		opts.PIDPath = opts.StatePath + ".watch.pid"
	}
	if opts.Notify == nil {
		opts.Notify = NotifyDesktop
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.Sleep == nil {
		opts.Sleep = time.Sleep
	}
	if opts.Poll <= 0 {
		opts.Poll = defaultWatchPoll
	}

	release, already, err := acquireWatchLock(opts.PIDPath)
	if err != nil {
		return err
	}
	if already {
		return nil
	}
	defer release()

	for {
		update, err := ClaimThresholds(opts.StatePath, opts.Now())
		if errors.Is(err, ErrNoTimer) {
			return nil
		}
		if err != nil {
			return err
		}
		s := update.State
		if s == nil {
			return nil
		}

		min, max, ok := s.Limits()
		if !ok {
			return nil
		}
		if update.Changed {
			_ = opts.Notify("Daylist", update.Message)
		}
		if s.MaxNotified {
			return nil
		}
		if s.Paused {
			opts.Sleep(opts.Poll)
			continue
		}

		delay := nextWatchDelay(min, max, s.TotalMsAt(opts.Now().UnixMilli()), s.MinNotified)
		if delay <= 0 {
			delay = opts.Poll
		}
		if delay > opts.Poll {
			delay = opts.Poll
		}
		opts.Sleep(delay)
	}
}

// nextWatchDelay devolve quanto falta para o próximo limite ainda não
// notificado.
func nextWatchDelay(min, max int, total int64, minNotified bool) time.Duration {
	target := min
	if minNotified {
		target = max
	}
	untilMs := int64(target)*60_000 - total
	if untilMs < 0 {
		return 0
	}
	return time.Duration(untilMs) * time.Millisecond
}

func acquireWatchLock(path string) (release func(), already bool, err error) {
	for attempt := 0; attempt < 3; attempt++ {
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err == nil {
			if _, writeErr := fmt.Fprintf(f, "%d\n", os.Getpid()); writeErr != nil {
				_ = f.Close()
				_ = os.Remove(path)
				return nil, false, writeErr
			}
			if closeErr := f.Close(); closeErr != nil {
				_ = os.Remove(path)
				return nil, false, closeErr
			}
			return func() { _ = os.Remove(path) }, false, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, false, err
		}

		pid, readErr := readPIDFile(path)
		if readErr == nil && processAlive(pid) {
			return nil, true, nil
		}
		if removeErr := os.Remove(path); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			return nil, false, removeErr
		}
	}
	return nil, false, errors.New("watch do timer: não foi possível obter o lock")
}

func readPIDFile(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		return 0, fmt.Errorf("pid inválido em %s", path)
	}
	return pid, nil
}

// StopWatch encerra o watch registrado no arquivo de PID, se ele ainda existir.
func StopWatch(pidPath string) {
	if pidPath == "" {
		return
	}
	pid, err := readPIDFile(pidPath)
	if err == nil && processAlive(pid) {
		signalWatchStop(pid)
		deadline := time.Now().Add(500 * time.Millisecond)
		for processAlive(pid) && time.Now().Before(deadline) {
			time.Sleep(10 * time.Millisecond)
		}
	}
	_ = os.Remove(pidPath)
}

// SpawnWatch inicia o watch em segundo plano, sem terminal e sem bloquear o
// processo atual. Se não houver timer ou se já existir um watch vivo, não faz
// nada.
func SpawnWatch(statePath, pidPath string) error {
	if statePath == "" {
		return ErrNoTimer
	}
	if pidPath == "" {
		pidPath = statePath + ".watch.pid"
	}
	if _, err := os.Stat(statePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrNoTimer
		}
		return err
	}
	if pid, err := readPIDFile(pidPath); err == nil && processAlive(pid) {
		return nil
	}

	exe, err := watchExecutable()
	if err != nil {
		return err
	}
	cmd := exec.Command(exe, "timer", "watch")
	cmd.Env = os.Environ()
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	detachProcess(cmd)
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
