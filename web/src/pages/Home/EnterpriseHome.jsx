/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React from 'react';
import { Link } from 'react-router-dom';
import {
  Activity,
  ArrowRight,
  Copy,
  Download,
  FileText,
  Gauge,
  Globe2,
  Headphones,
  Layers3,
  Link2,
  Network,
  Play,
  Server,
  ShieldCheck,
  Sparkles,
  Users,
  Wallet,
  Zap,
} from 'lucide-react';
import {
  Claude,
  DeepSeek,
  Gemini,
  Grok,
  Midjourney,
  Mistral,
  OpenAI,
  Qwen,
} from '@lobehub/icons';
import './EnterpriseHome.css';

const providerCards = [
  { name: 'OpenAI', Icon: OpenAI },
  { name: 'Claude', Icon: Claude, color: true },
  { name: 'Gemini', Icon: Gemini, color: true },
  { name: 'Grok', Icon: Grok, color: true },
  { name: 'DeepSeek', Icon: DeepSeek, color: true },
  { name: 'Qwen', Icon: Qwen, color: true },
  { name: 'Midjourney', Icon: Midjourney },
  { name: 'Mistral', Icon: Mistral, color: true },
];

const modelTags = [
  'GPT-5 / GPT-4o',
  'Claude Sonnet',
  'Gemini Pro',
  'DeepSeek',
  'Qwen',
  'Midjourney',
  'Codex',
  'Claude Code',
];

const featureItems = [
  ['企业级高速通道', '多供应商、多账号、多线路汇聚，优先保证请求稳定落地，适合高频调用场景。', Server],
  ['协议完全兼容', '兼容 OpenAI、Claude、Gemini 常见接口形态，现有 SDK 和客户端低成本迁移。', Link2],
  ['智能负载均衡', '按模型、分组、倍率、渠道健康度与失败状态动态调度，减少单点波动。', Network],
  ['透明计费', 'Token、缓存、图片、按次任务统一折算，余额、日志与账单链路可追踪。', Wallet],
  ['多节点容灾', '线路异常时自动切换与重试，结合渠道健康度降低超时、断流和上游波动影响。', Globe2],
  ['7×24 服务保障', '面向团队、代理和商业用户提供持续支持，关键问题可快速定位到模型、渠道或用户。', Headphones],
  ['多模态覆盖', '文本、视觉、图片生成、Claude Code、Codex、Gemini CLI 等场景一站接入。', Sparkles],
  ['代理与团队运营', '支持分组、密钥、额度、倍率与用量视图，方便代理加盟和团队级成本管理。', Users],
];

const quickSteps = [
  ['01', '注册并获取密钥', '登录控制台创建 API Key，按团队、用户或项目设置额度、可用模型和过期时间。'],
  ['02', '替换模型基址', '把原客户端的 Base URL 改成元衡 API 地址，保留原有 SDK、参数和调用结构。'],
  ['03', '稳定开始调用', '通过日志、账单、渠道状态和模型广场持续观测请求表现，按需切换策略。'],
];

const ProviderLogo = ({ Icon, color = false }) => {
  const Logo = color && Icon.Color ? Icon.Color : Icon;
  return <Logo size={24} />;
};

const splitProviders = (items) => [items.slice(0, 4), items.slice(4)];

const EnterpriseHome = ({
  brandName,
  brandLogo,
  currentEndpoint,
  endpointItems,
  handleCopyEndpoint,
  handleOpenDocs,
  serverAddress,
  setEndpointIndex,
  t,
}) => {
  const [leftProviders, rightProviders] = splitProviders(providerCards);

  const docsAction = handleOpenDocs ? (
    <button type='button' className='yh-btn yh-btn-secondary' onClick={handleOpenDocs}>
      <FileText size={18} />
      {t('阅读 API 文档')}
    </button>
  ) : (
    <Link className='yh-btn yh-btn-secondary' to='/docs'>
      <FileText size={18} />
      {t('阅读 API 文档')}
    </Link>
  );

  return (
    <div className='home-enterprise'>
      <section className='yh-page-shell yh-hero' aria-labelledby='hero-title'>
        <div className='yh-hero-copy'>
          <div className='yh-eyebrow'>{t('元起万模，衡定全局')}</div>
          <h1 id='hero-title'>
            {t('元衡 API：一口接入多模型，')}
            <span>{t('智能调度更稳定')}</span>
          </h1>
          <p className='yh-slogan'>{t('更强模型，更低价格，更易落地。')}</p>
          <p className='yh-lead'>
            {t('统一接入 OpenAI、Claude、Gemini、DeepSeek、Qwen、Midjourney 等主流模型，通过智能路由与负载均衡，为企业提供高可用、低延迟、透明计费的全托管 AI API 网关服务，助力业务快速集成与规模化落地。')}
          </p>

          <div className='yh-endpoint' aria-label='API 接入地址'>
            <span className='yh-endpoint-main'>
              <Globe2 size={18} />
              <code>{serverAddress}</code>
            </span>
            <span className='yh-endpoint-divider' aria-hidden='true' />
            <select
              value={currentEndpoint}
              onChange={(event) => {
                const nextIndex = endpointItems.findIndex(
                  (item) => item.value === event.target.value,
                );
                if (nextIndex >= 0) setEndpointIndex(nextIndex);
              }}
              aria-label='选择 API endpoint'
            >
              {endpointItems.map((item) => (
                <option key={item.value} value={item.value}>
                  {item.value}
                </option>
              ))}
            </select>
            <button
              type='button'
              className='yh-copy-btn'
              onClick={handleCopyEndpoint}
              aria-label={t('复制完整地址')}
              title={t('复制完整地址')}
            >
              <Copy size={18} />
            </button>
          </div>

          <div className='yh-hero-actions'>
            <Link className='yh-btn yh-btn-primary' to='/console'>
              {t('开启 AI 新体验')}
              <ArrowRight size={18} />
            </Link>
            <Link className='yh-btn yh-btn-secondary' to='/download'>
              <Download size={18} />
              {t('下载客户端')}
            </Link>
          </div>

          <div className='yh-proof' aria-label='平台能力指标'>
            <div className='yh-proof-item'>
              <span><Layers3 size={22} /></span>
              <b>500+</b>
              <small>{t('已接入模型')}</small>
            </div>
            <div className='yh-proof-item'>
              <span><ShieldCheck size={22} /></span>
              <b>99.9%</b>
              <small>{t('稳定性保障')}</small>
            </div>
            <div className='yh-proof-item'>
              <span><Headphones size={22} /></span>
              <b>7×24</b>
              <small>{t('服务支持')}</small>
            </div>
          </div>
        </div>

        <aside className='yh-gateway-panel' aria-label='AI Gateway 调度面板示意'>
          <div className='yh-panel-top'>
            <div>
              <strong>AI Gateway</strong>
              <span className='yh-status'>{t('运行正常')}</span>
            </div>
            <span className='yh-period'>{t('近 24 小时')}</span>
          </div>

          <div className='yh-routing-board'>
            <div className='yh-provider-col'>
              {leftProviders.map(({ name, Icon, color }) => (
                <div className='yh-provider-card' key={name}>
                  <span className='yh-provider-logo'>
                    <ProviderLogo Icon={Icon} color={color} />
                  </span>
                  <strong>{name}</strong>
                  <i />
                </div>
              ))}
            </div>

            <div className='yh-orchestration'>
              <svg className='yh-network-svg' viewBox='0 0 400 290' fill='none' aria-hidden='true'>
                <defs>
                  <linearGradient id='yhRouteLine' x1='54' y1='30' x2='336' y2='246' gradientUnits='userSpaceOnUse'>
                    <stop stopColor='#125CFF' stopOpacity='0.92' />
                    <stop offset='1' stopColor='#0ECDD7' stopOpacity='0.9' />
                  </linearGradient>
                  <linearGradient id='yhHexFill' x1='120' y1='70' x2='280' y2='220' gradientUnits='userSpaceOnUse'>
                    <stop stopColor='#125CFF' stopOpacity='0.11' />
                    <stop offset='1' stopColor='#0ECDD7' stopOpacity='0.08' />
                  </linearGradient>
                </defs>
                <g className='yh-hex-spin'>
                  <path d='M200 34 312 97v126L200 286 88 223V97L200 34Z' fill='url(#yhHexFill)' stroke='rgba(18,92,255,.2)' />
                  <path d='M200 64 278 108v88l-78 44-78-44v-88l78-44Z' stroke='rgba(18,92,255,.18)' />
                  <path d='M200 93 245 119v52l-45 26-45-26v-52l45-26Z' stroke='rgba(14,205,215,.22)' />
                </g>
                <path className='yh-flow' d='M22 82c56 0 74 60 142 60' stroke='url(#yhRouteLine)' strokeWidth='2.2' strokeLinecap='round' strokeDasharray='4 7' />
                <path className='yh-flow' d='M22 145h140' stroke='url(#yhRouteLine)' strokeWidth='2.2' strokeLinecap='round' strokeDasharray='4 7' />
                <path className='yh-flow' d='M22 208c56 0 74-60 142-60' stroke='url(#yhRouteLine)' strokeWidth='2.2' strokeLinecap='round' strokeDasharray='4 7' />
                <path className='yh-flow' d='M378 82c-56 0-74 60-142 60' stroke='url(#yhRouteLine)' strokeWidth='2.2' strokeLinecap='round' strokeDasharray='4 7' />
                <path className='yh-flow' d='M378 145H238' stroke='url(#yhRouteLine)' strokeWidth='2.2' strokeLinecap='round' strokeDasharray='4 7' />
                <path className='yh-flow' d='M378 208c-56 0-74-60-142-60' stroke='url(#yhRouteLine)' strokeWidth='2.2' strokeLinecap='round' strokeDasharray='4 7' />
                {[82, 145, 208].map((cy) => <circle className='yh-pulse-node' key={`l-${cy}`} cx='22' cy={cy} r='5' fill='#125CFF' />)}
                {[82, 145, 208].map((cy) => <circle className='yh-pulse-node' key={`r-${cy}`} cx='378' cy={cy} r='5' fill='#0ECDD7' />)}
              </svg>
              <span className='yh-hub-rings' />
              <span className='yh-hub-core'>
                <img src={brandLogo} alt='' />
              </span>
              <span className='yh-hub-label'>{brandName} 智能调度层</span>
            </div>

            <div className='yh-provider-col'>
              {rightProviders.map(({ name, Icon, color }) => (
                <div className='yh-provider-card' key={name}>
                  <span className='yh-provider-logo'>
                    <ProviderLogo Icon={Icon} color={color} />
                  </span>
                  <strong>{name}</strong>
                  <i />
                </div>
              ))}
            </div>
          </div>

          <div className='yh-metric-grid'>
            <div className='yh-metric-card'>
              <small>{t('稳定可用')}</small>
              <b>99.9%</b>
              <div className='yh-ring' />
            </div>
            <div className='yh-metric-card'>
              <small>{t('智能路由')}</small>
              <b>Live</b>
              <div className='yh-bars' aria-hidden='true'><span /><span /><span /><span /><span /></div>
            </div>
            <div className='yh-metric-card'>
              <small>{t('透明计费')}</small>
              <b>Token</b>
              <svg className='yh-sparkline' viewBox='0 0 160 42' aria-hidden='true'><path d='M3 32 C28 20, 40 26, 58 15 S92 14, 112 10 S138 13, 156 4' fill='none' stroke='#125cff' strokeWidth='3' strokeLinecap='round' /></svg>
            </div>
            <div className='yh-metric-card'>
              <small>{t('策略状态')}</small>
              <b>OK</b>
              <div className='yh-strategy-flow'><Activity size={42} /></div>
            </div>
          </div>
        </aside>
      </section>

      <section className='yh-page-shell yh-section' id='features'>
        <div className='yh-section-head'>
          <div>
            <span className='yh-section-label'>ADVANTAGES</span>
            <h2>{t('我们的优势')}</h2>
          </div>
          <p>{t('把模型多、接入快、可部署、可观测放到同一套控制平面，首页表达更接近企业级 API 网关。')}</p>
        </div>
        <div className='yh-feature-grid'>
          {featureItems.map(([title, desc, Icon]) => (
            <article className='yh-feature-card' key={title}>
              <span className='yh-card-icon'><Icon size={25} strokeWidth={1.9} /></span>
              <h3>{t(title)}</h3>
              <p>{t(desc)}</p>
            </article>
          ))}
        </div>
      </section>

      <section className='yh-page-shell yh-section' id='models'>
        <div className='yh-section-head'>
          <div>
            <span className='yh-section-label'>MODEL HUB</span>
            <h2>{t('一个入口，覆盖多种客户端')}</h2>
          </div>
          <p>{t('统一 Base URL 与 API Key，覆盖 OpenAI-compatible、Claude、Gemini、Codex、Claude Code 等接入方式。')}</p>
        </div>
        <div className='yh-model-strip'>
          <article className='yh-model-card'>
            <h3>{t('500+ 模型能力池')}</h3>
            <p>{t('从通用对话、代码编程、长文本、图像生成到多模态理解，都可以通过统一入口接入。')}</p>
            <div className='yh-model-tags'>
              {modelTags.map((tag) => <span key={tag}>{tag}</span>)}
            </div>
          </article>
          <article className='yh-model-card yh-terminal-card'>
            <div className='yh-terminal-line'><b>Base URL</b><span>{serverAddress}</span></div>
            <div className='yh-terminal-line'><b>Endpoint</b><span>{currentEndpoint}</span></div>
            <div className='yh-terminal-line'><b>Client</b><span>Codex / Claude Code / Gemini CLI</span></div>
            <div className='yh-terminal-line'><b>Routing</b><span>healthy · balanced · observable</span></div>
            <div className='yh-terminal-line'><b>Billing</b><span>token · cache · image · per-call</span></div>
          </article>
        </div>
      </section>

      <section className='yh-page-shell yh-section' id='quick-start'>
        <div className='yh-section-head'>
          <div>
            <span className='yh-section-label'>QUICK START</span>
            <h2>{t('三步快速接入')}</h2>
          </div>
          <p>{t('不用改变你的 SDK 习惯，只替换 Base URL 和 Key，即可把多模型调用切到元衡 API。')}</p>
        </div>
        <div className='yh-quick-start'>
          {quickSteps.map(([n, title, desc]) => (
            <article className='yh-step-card' data-step={n} key={n}>
              <span>{n}</span>
              <h3>{t(title)}</h3>
              <p>{t(desc)}</p>
            </article>
          ))}
        </div>
      </section>

      <section className='yh-page-shell yh-always-on' id='support'>
        <div className='yh-support-banner'>
          <h2><span>24/7/365</span> {t('全天候支持')}</h2>
          <p className='yh-support-subtitle'>{t('我们时刻恭候您')}</p>
          <p>{t('从模型选型、客户端接入、代理运营到异常排查，元衡 API 都提供更清晰的控制台和更稳定的调用链路。')}</p>
          <div className='yh-support-actions'>
            <Link className='yh-btn yh-btn-primary' to='/console'>
              <Play size={18} />
              {t('立即进入控制台')}
            </Link>
            {docsAction}
          </div>
        </div>
      </section>
    </div>
  );
};

export default EnterpriseHome;
