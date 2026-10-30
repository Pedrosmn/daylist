module.exports = async (params) => {
  const { app } = params;
  const moment = window.moment;

  const weeklistsRoot = "Fleeting Notes/Daily/weeklists";
  const lineRe = /^- \[([ xX])\] (.+?) \((.*?)\) - (.*)$/;

  // Semana-alvo (a nova weeklist a ser criada). Por padrão é a semana
  // seguinte à atual, mas `daylist -w` pode ter deixado um arquivo na raiz
  // do vault indicando a semana atual em vez disso — usado quando a semana
  // atual ainda não tem weeklist.md (recuperação de uma semana perdida).
  const targetFilePath = ".daylist-weeklist-target.json";
  let targetMoment = moment().add(1, "weeks");

  if (await app.vault.adapter.exists(targetFilePath)) {
    try {
      const raw = await app.vault.adapter.read(targetFilePath);
      const parsed = JSON.parse(raw);
      const match = (parsed.week || "").match(/^(\d{4})-W(\d{2})$/);
      if (match) {
        targetMoment = moment().isoWeekYear(parseInt(match[1], 10)).isoWeek(parseInt(match[2], 10));
      }
    } catch (e) {
      new Notice("Aviso: não foi possível ler a semana-alvo; usando a semana seguinte.");
    } finally {
      await app.vault.adapter.remove(targetFilePath);
    }
  }

  const nextWeekFolder = targetMoment.format("GGGG-[W]WW");
  const nextWeeklistFolder = `${weeklistsRoot}/${nextWeekFolder}`;
  const nextWeeklistPath = `${nextWeeklistFolder}/weeklist.md`;

  if (app.vault.getAbstractFileByPath(nextWeeklistPath)) {
    new Notice(`Weeklist já existe: ${nextWeeklistPath}`);
    return;
  }

  // Semana-fonte: a semana imediatamente anterior à semana-alvo, de onde os
  // hábitos são copiados (reset de horário/flag, mantendo a lista fixa).
  const sourceMoment = targetMoment.clone().subtract(1, "weeks");
  const currentWeekFolder = sourceMoment.format("GGGG-[W]WW");
  const currentWeeklistPath = `${weeklistsRoot}/${currentWeekFolder}/weeklist.md`;

  let habitosTexto = "";
  const currentWeeklistFile = app.vault.getAbstractFileByPath(currentWeeklistPath);
  if (currentWeeklistFile) {
    const content = await app.vault.read(currentWeeklistFile);
    const lines = content.split("\n");
    let inHab = false;
    const habLines = [];
    for (const line of lines) {
      if (line.startsWith("# Hábitos")) { inHab = true; continue; }
      if (inHab && line.startsWith("# ")) { inHab = false; continue; }
      if (inHab) {
        if (line.startsWith("## ")) { habLines.push(line); continue; }
        const m = line.match(lineRe);
        if (m) habLines.push(`- [ ] xST () - ${m[4].trim()}`);
      }
    }
    habitosTexto = habLines.join("\n");
  }

  if (!app.vault.getAbstractFileByPath(nextWeeklistFolder)) {
    await app.vault.createFolder(nextWeeklistFolder);
  }

  const novaWeeklist = `# Tasks\n\n---\n# Hábitos\n${habitosTexto ? habitosTexto + "\n" : ""}`;
  await app.vault.create(nextWeeklistPath, novaWeeklist);
  new Notice(`Weeklist criada: ${nextWeeklistPath}`);
};
