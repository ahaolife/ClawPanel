import { api } from './api';

export interface OpenClawInstallPrereqStatus {
  ok: boolean;
  requiresManualInstall: boolean;
  message: string;
  nodeUrl: string;
  gitUrl: string;
}

function nodeMajor(version: string): number {
  const raw = String(version || '').trim().replace(/^v/i, '');
  const major = Number(raw.split('.')[0]);
  return Number.isFinite(major) ? major : -1;
}

export async function getOpenClawInstallPrerequisiteStatus(): Promise<OpenClawInstallPrereqStatus> {
  const base = {
    ok: true,
    requiresManualInstall: false,
    message: '',
    nodeUrl: 'https://nodejs.org',
    gitUrl: 'https://git-scm.com/downloads',
  };
  try {
    const r = await api.getSoftwareList();
    if (!r?.ok) return base;
    const platform = String(r.platform || '').toLowerCase();
    if (platform !== 'windows' && platform !== 'darwin') return base;
    const software = Array.isArray(r.software) ? r.software : [];
    const node = software.find((s: any) => s.id === 'nodejs');
    const git = software.find((s: any) => s.id === 'git');
    const missing: string[] = [];
    const nodeVersion = String(node?.version || '').trim();
    if (!node?.installed || !nodeVersion) missing.push('Node.js (>=20)');
    else if (nodeMajor(nodeVersion) < 20) missing.push(`Node.js >=20（当前 ${nodeVersion}）`);
    if (!git?.installed || !String(git?.version || '').trim()) missing.push('Git');
    if (missing.length === 0) return base;
    const platformLabel = platform === 'windows' ? 'Windows' : 'macOS';
    return {
      ...base,
      ok: false,
      requiresManualInstall: true,
      message: `检测到 ${platformLabel} 缺少 ${missing.join(' 和 ')}。为避免一键安装中途报错，请先手动安装 Node.js 与 Git，再回来继续安装 OpenClaw。`,
    };
  } catch {
    return base;
  }
}

export async function ensureOpenClawInstallPrerequisites(): Promise<OpenClawInstallPrereqStatus> {
  return getOpenClawInstallPrerequisiteStatus();
}
