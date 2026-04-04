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
import React, { useState } from 'react';
import {
  Button,
  Collapsible,
  Space,
  Switch,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import {
  AlertTriangle,
  ChevronDown,
  ChevronRight,
  Info,
  RefreshCw,
  Save,
  Wallet,
} from 'lucide-react';
import { getWalletStatusMeta } from '../utils';

const { Text, Title } = Typography;

const ProfitBoardHeader = ({
  querying,
  overviewQuerying,
  runFullRefresh,
  saving,
  saveConfig,
  autoRefreshMode,
  setAutoRefreshMode,
  statusSummary,
  hasNewActivity,
  generatedAtText,
  combinedWarnings,
  sitePriceFactorNote,
  walletModeEnabled,
  selectedAccount,
  t,
}) => {
  const [warningsExpanded, setWarningsExpanded] = useState(false);
  const allMessages = [
    ...combinedWarnings.map((w) => ({ type: 'warning', text: w })),
    ...(sitePriceFactorNote
      ? [{ type: 'info', text: sitePriceFactorNote }]
      : []),
  ];
  const hasMessages = allMessages.length > 0;

  return (
    <>
      <div className='flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between'>
        <div className='flex items-center gap-3'>
          <Title heading={4} style={{ marginBottom: 0 }}>
            {t('收益看板')}
          </Title>
          <Text type='tertiary' size='small'>
            {generatedAtText}
          </Text>
          {hasNewActivity && (
            <Tag color='orange' size='small'>
              {t('有新数据')}
            </Tag>
          )}
          {hasMessages && (
            <Tag
              color='amber'
              size='small'
              className='cursor-pointer'
              onClick={() => setWarningsExpanded(!warningsExpanded)}
            >
              {combinedWarnings.length} {t('个问题')}
            </Tag>
          )}
        </div>
        <Space wrap>
          <Button
            theme='solid'
            type='primary'
            icon={<RefreshCw size={14} />}
            loading={querying || overviewQuerying}
            onClick={runFullRefresh}
            size='small'
          >
            {t('刷新')}
          </Button>
          <Button
            theme='solid'
            type='tertiary'
            icon={<Save size={14} />}
            loading={saving}
            onClick={saveConfig}
            size='small'
          >
            {t('保存配置')}
          </Button>
          <div className='flex items-center gap-1.5 rounded-full border border-semi-color-border bg-semi-color-fill-0 px-2.5 py-1'>
            <Text type='tertiary' size='small'>
              {t('自动检查')}
            </Text>
            <Switch
              checked={autoRefreshMode}
              onChange={setAutoRefreshMode}
              size='small'
            />
          </div>
        </Space>
      </div>

      {hasMessages && (
        <Collapsible isOpen={warningsExpanded}>
          <div className='mt-2 space-y-0.5 rounded-lg bg-semi-color-fill-0 px-3 py-2'>
            {allMessages.map((msg, idx) => (
              <div
                key={idx}
                className='flex items-start gap-2 py-1 text-sm'
              >
                {msg.type === 'warning' ? (
                  <AlertTriangle
                    size={13}
                    className='mt-0.5 shrink-0 text-semi-color-warning'
                  />
                ) : (
                  <Info
                    size={13}
                    className='mt-0.5 shrink-0 text-semi-color-primary'
                  />
                )}
                <span className='text-semi-color-text-1'>{msg.text}</span>
              </div>
            ))}
          </div>
        </Collapsible>
      )}
    </>
  );
};

export default ProfitBoardHeader;
