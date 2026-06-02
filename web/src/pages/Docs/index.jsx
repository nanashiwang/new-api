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

import React, { useMemo } from 'react';
import { Button, Card, Space, Typography } from '@douyinfe/semi-ui';
import { IconBookOpenStroked, IconLink } from '@douyinfe/semi-icons';
import { useLocation, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import MarkdownRenderer from '../../components/common/markdown/MarkdownRenderer';
import usageMarkdown from '../../../../docs/NAN_USAGE.md?raw';
import clientsMarkdown from '../../../../docs/NAN_CLIENTS.md?raw';
import troubleshootingMarkdown from '../../../../docs/NAN_TROUBLESHOOTING.md?raw';
import './index.css';

const { Title, Text } = Typography;
const DOCS_EYEBROW = 'NAN Docs';
const CARD_SHADOWS = 'always';

const DOCS = [
  {
    key: 'usage',
    path: '/docs',
    title: '平台教程',
    description: '注册、令牌、调用、充值、发票、扣费和安全提醒',
    content: usageMarkdown,
  },
  {
    key: 'clients',
    path: '/docs/clients',
    title: '客户端接入',
    description: 'Codex、Claude Code、Gemini、OpenCode、OpenClaw、CC Switch',
    content: clientsMarkdown,
  },
  {
    key: 'troubleshooting',
    path: '/docs/troubleshooting',
    title: '常见报错排查',
    description: '按错误文本排查注册、认证、模型、客户端、网络、支付问题',
    content: troubleshootingMarkdown,
  },
];

const DOC_LINK_REPLACEMENTS = [
  [/\.\/NAN_USAGE\.md/g, '/docs'],
  [/\.\/NAN_CLIENTS\.md/g, '/docs/clients'],
  [/\.\/NAN_TROUBLESHOOTING\.md/g, '/docs/troubleshooting'],
];

function normalizeMarkdownLinks(content) {
  return DOC_LINK_REPLACEMENTS.reduce(
    (current, [pattern, replacement]) => current.replace(pattern, replacement),
    content,
  );
}

function getActiveDoc(pathname) {
  const matched = DOCS.find(
    (doc) => doc.path !== '/docs' && pathname.startsWith(doc.path),
  );
  return matched || DOCS[0];
}

export default function DocsPage() {
  const { t } = useTranslation();
  const location = useLocation();
  const navigate = useNavigate();
  const activeDoc = getActiveDoc(location.pathname);
  const content = useMemo(
    () => normalizeMarkdownLinks(activeDoc.content),
    [activeDoc.content],
  );

  return (
    <div className='nan-docs-page'>
      <section className='nan-docs-hero'>
        <div>
          <Text className='nan-docs-eyebrow'>{DOCS_EYEBROW}</Text>
          <Title heading={2} className='nan-docs-title'>
            {t('平台使用文档')}
          </Title>
          <Text type='tertiary'>
            {t('新手先看平台教程和配置速查；遇到错误直接查常见报错排查。')}
          </Text>
        </div>
        <Button
          icon={<IconLink />}
          theme='borderless'
          onClick={() =>
            window.open('https://docs.newapi.pro/zh/docs', '_blank')
          }
        >
          {t('上游文档')}
        </Button>
      </section>

      <div className='nan-docs-layout'>
        <aside className='nan-docs-sidebar'>
          <Space vertical align='start' style={{ width: '100%' }}>
            {DOCS.map((doc) => (
              <Button
                key={doc.key}
                block
                icon={<IconBookOpenStroked />}
                theme={doc.key === activeDoc.key ? 'solid' : 'borderless'}
                type={doc.key === activeDoc.key ? 'primary' : 'tertiary'}
                className='nan-docs-nav-button'
                onClick={() => navigate(doc.path)}
              >
                {t(doc.title)}
              </Button>
            ))}
          </Space>
        </aside>

        <main className='nan-docs-content'>
          <Card className='nan-docs-card' shadows={CARD_SHADOWS}>
            <div className='nan-docs-card-header'>
              <div>
                <Title heading={3}>{t(activeDoc.title)}</Title>
                <Text type='tertiary'>{t(activeDoc.description)}</Text>
              </div>
            </div>
            <MarkdownRenderer content={content} fontSize={15} />
          </Card>
        </main>
      </div>
    </div>
  );
}
