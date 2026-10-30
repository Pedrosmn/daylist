module.exports = async (params) => {
  const { app } = params;
  const moment = window.moment;

  const daysFolder = "Fleeting Notes/Daily/Days";
  const weeklistsRoot = "Fleeting Notes/Daily/weeklists";
  const lineRe = /^- \[([ xX])\] (.+?) \((.*?)\) - (.*)$/;

  // Remove o sufixo do timer (ex: " ⏱ 251min (14:36–17:54)") de uma descrição,
  // para que o mesmo hábito não seja contado como um item separado quando o
  // timer foi usado naquele dia.
  const stripTimerSuffix = (desc) => desc.replace(/\s*⏱.*$/u, "").trim();

  // Semana-alvo: por padrão a semana atual, mas `daylist -f` pode ter
  // deixado um arquivo na raiz do vault indicando outra semana (ex: quando
  // a semana anterior ainda não tem feedback.md e o usuário escolheu gerar
  // aquela em vez da atual).
  const targetFilePath = ".daylist-feedback-target.json";
  let currentWeekFolder = moment().format("GGGG-[W]WW");
  let targetMoment = moment();

  if (await app.vault.adapter.exists(targetFilePath)) {
    try {
      const raw = await app.vault.adapter.read(targetFilePath);
      const parsed = JSON.parse(raw);
      const match = (parsed.week || "").match(/^(\d{4})-W(\d{2})$/);
      if (match) {
        currentWeekFolder = parsed.week;
        targetMoment = moment().isoWeekYear(parseInt(match[1], 10)).isoWeek(parseInt(match[2], 10));
      }
    } catch (e) {
      new Notice("Aviso: não foi possível ler a semana-alvo; usando a semana atual.");
    } finally {
      await app.vault.adapter.remove(targetFilePath);
    }
  }

  const currentWeeklistPath = `${weeklistsRoot}/${currentWeekFolder}/weeklist.md`;
  const feedbackPath = `${weeklistsRoot}/${currentWeekFolder}/feedback.md`;

  const currentWeeklistFile = app.vault.getAbstractFileByPath(currentWeeklistPath);
  if (!currentWeeklistFile) {
    new Notice(`Weeklist não encontrada em: ${currentWeeklistPath}`);
    return;
  }
  if (app.vault.getAbstractFileByPath(feedbackPath)) {
    new Notice(`Feedback já existe: ${feedbackPath}`);
    return;
  }

  const weeklistContent = await app.vault.read(currentWeeklistFile);

  const habitStats = new Map();
  const categoryOrder = [];
  {
    const wLines = weeklistContent.split("\n");
    let inHab = false, cat = null;
    for (const line of wLines) {
      if (line.startsWith("# Hábitos")) { inHab = true; cat = null; continue; }
      if (inHab && line.startsWith("# ")) { inHab = false; continue; }
      if (inHab && line.startsWith("## ")) {
        cat = line.replace("## ", "").trim();
        if (!categoryOrder.includes(cat)) categoryOrder.push(cat);
        continue;
      }
      if (inHab) {
        const m = line.match(lineRe);
        if (m) {
          const desc = stripTimerSuffix(m[4].trim());
          const key = `${cat}||${desc}`;
          if (!habitStats.has(key)) habitStats.set(key, { category: cat, desc, checked: 0, total: 0 });
        }
      }
    }
  }

  const startOfWeek = targetMoment.clone().startOf("isoWeek");
  for (let i = 0; i < 7; i++) {
    const dateStr = startOfWeek.clone().add(i, "days").format("YYYY-MM-DD");
    const dayFile = app.vault.getAbstractFileByPath(`${daysFolder}/${dateStr}.md`);
    if (!dayFile) continue;

    const content = await app.vault.read(dayFile);
    const lines = content.split("\n");
    let inHabitos = false, currentCategory = null;

    for (const line of lines) {
      if (line.startsWith("# Hábitos")) { inHabitos = true; currentCategory = null; continue; }
      if (inHabitos && line.startsWith("# ")) { inHabitos = false; continue; }
      if (inHabitos && line.startsWith("## ")) { currentCategory = line.replace("## ", "").trim(); continue; }
      if (inHabitos) {
        const m = line.match(lineRe);
        if (m) {
          const desc = stripTimerSuffix(m[4].trim());
          const key = `${currentCategory}||${desc}`;
          if (!habitStats.has(key)) {
            if (!categoryOrder.includes(currentCategory)) categoryOrder.push(currentCategory);
            habitStats.set(key, { category: currentCategory, desc, checked: 0, total: 0 });
          }
          const stat = habitStats.get(key);
          stat.total++;
          if (m[1].trim() !== "") stat.checked++;
        }
      }
    }
  }

  let resumoTexto;
  if (habitStats.size === 0) {
    resumoTexto = "Nenhum hábito encontrado na weeklist.\n";
  } else {
    let totalChecked = 0, totalHabitos = 0;
    for (const s of habitStats.values()) { totalChecked += s.checked; totalHabitos += s.total; }
    const pctGeral = totalHabitos > 0 ? Math.round((totalChecked / totalHabitos) * 100) : 0;

    resumoTexto = `Hábitos concluídos na semana: ${totalChecked}/${totalHabitos} (${pctGeral}%)\n`;
    for (const cat of categoryOrder) {
      resumoTexto += `\n## ${cat}\n`;
      for (const s of [...habitStats.values()].filter(s => s.category === cat)) {
        const pct = s.total > 0 ? Math.round((s.checked / s.total) * 100) : 0;
        resumoTexto += `${s.desc} ${s.checked}/${s.total} - (${pct}%)\n`;
      }
    }
  }

  let feedbackConteudo = `# Feedback - Semana ${currentWeekFolder}\n`;
  feedbackConteudo += `Criado em: ${moment().format("YYYY-MM-DD HH:mm")}\n\n`;
  for (const cat of categoryOrder) {
    feedbackConteudo += `## ${cat}\n\n`;
  }
  feedbackConteudo += `---\n# Resumo\n${resumoTexto}`;

  await app.vault.create(feedbackPath, feedbackConteudo);
  new Notice(`Feedback criado: ${feedbackPath}`);
};
