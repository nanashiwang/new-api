export const formatContentSafetyCategory = (category, t) => {
  const code = typeof category === 'string' ? category.trim() : '';
  if (!code) return '-';
  const labels = {
    credential_theft_phishing: t('钓鱼或凭证窃取'),
    malware: t('恶意软件'),
    ransomware: t('勒索软件'),
    unauthorized_access: t('未授权访问'),
    exploit_development: t('漏洞利用开发'),
    privilege_escalation: t('权限提升'),
    persistence_backdoor: t('持久化或后门'),
    security_evasion: t('安全规避'),
    data_exfiltration: t('数据窃取或外传'),
    scanning_reconnaissance: t('攻击性扫描或侦察'),
    ddos_botnet: t('DDoS 或僵尸网络'),
    automated_abuse: t('自动化滥用'),
    account_takeover: t('账号接管'),
    child_sexual_content: t('未成年人性内容'),
    adult_sexual_content: t('成人露骨性内容'),
    violence_gore: t('暴力或血腥内容'),
    self_harm: t('自残或自杀'),
    hate_discrimination: t('仇恨或歧视'),
    harassment_threats: t('骚扰或威胁'),
    extremism: t('极端主义'),
    fraud_impersonation: t('欺诈或冒充'),
    privacy_doxxing: t('隐私侵犯或人肉'),
    illicit_regulated_goods: t('非法或受管制物品'),
    cyber_policy_other: t('其他网络安全高风险'),
    safety_policy_other: t('其他内容安全'),
  };
  const label = labels[code];
  return label ? `${label} (${code})` : code;
};
