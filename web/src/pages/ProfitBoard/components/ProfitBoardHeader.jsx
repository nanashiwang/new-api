import React, { useState } from 'react';
import {
  Button,
  Card,
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
  sharedSiteModelCount,
  warningSummary,
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
        <Title heading={4} style={{ marginBottom: 0 }}>
          {t('收益看板')}
        </Title>
        <Space wrap>
          <Button
            theme='solid'
            type='primary'
            icon={<RefreshCw size={14} />}
            loading={querying || overviewQuerying}
            onClick={runFullRefresh}
            size='default'
          >
            {t('刷新收益看板')}
          </Button>
          <Button
            theme='solid'
            type='tertiary'
            icon={<Save size={14} />}
            loading={saving}
            onClick={saveConfig}
            size='default'
          >
            {t('保存配置')}
          </Button>
          <div className='flex items-center gap-1.5 rounded-full border border-semi-color-border bg-semi-color-fill-0 px-2.5 py-1.5'>
            <Text type='tertiary' size='small'>
              {t('低频检查')}
            </Text>
            <Switch
              checked={autoRefreshMode}
              onChange={setAutoRefreshMode}
              size='small'
            />
          </div>
        </Space>
      </div>

      <Card bordered={false} bodyStyle={{ padding: '12px 16px' }}>
        <div
          className={`grid gap-3 ${walletModeEnabled ? 'xl:grid-cols-4' : 'xl:grid-cols-3'}`}
        >
          <div className='rounded-lg bg-semi-color-fill-0 px-3 py-2'>
            <Text type='tertiary' size='small'>
              {t('当前状态')}
            </Text>
            <div className='mt-1.5 flex flex-wrap gap-1.5'>
              {statusSummary.length > 0 ? (
                statusSummary.map((item) => (
                  <Tag key={item.key} color={item.color} size='small'>
                    {item.text}
                  </Tag>
                ))
              ) : (
                <Tag color='grey' size='small'>
                  {t('等待首次刷新')}
                </Tag>
              )}
              {hasNewActivity ? (
                <Tag color='orange' size='small'>
                  {t('检测到新数据，请手动刷新')}
                </Tag>
              ) : null}
            </div>
          </div>
          <div className='rounded-lg bg-semi-color-fill-0 px-3 py-2'>
            <Text type='tertiary' size='small'>
              {t('最近刷新')}
            </Text>
            <div className='mt-1.5 text-sm font-semibold'>
              {generatedAtText}
            </div>
          </div>
          <div className='rounded-lg bg-semi-color-fill-0 px-3 py-2'>
            <Text type='tertiary' size='small'>
              {t('问题摘要')}
            </Text>
            <div className='mt-1.5 text-sm font-semibold'>{warningSummary}</div>
          </div>
          <div className='rounded-lg bg-semi-color-fill-0 px-3 py-2'>
            <Text type='tertiary' size='small'>
              {t('配置概况')}
            </Text>
            <div className='mt-1.5 text-sm font-semibold'>
              {sharedSiteModelCount > 0
                ? t('共享定价已绑定 {{count}} 个组合', {
                    count: sharedSiteModelCount,
                  })
                : t('当前未启用共享定价')}
            </div>
          </div>
          {walletModeEnabled ? (
            <div className='rounded-lg bg-semi-color-fill-0 px-3 py-2'>
              <div className='flex items-center gap-1.5'>
                <Wallet size={13} />
                <Text type='tertiary' size='small'>
                  {t('上游钱包')}
                </Text>
              </div>
              <div className='mt-1.5 truncate text-sm font-semibold'>
                {selectedAccount?.name || t('未选择账户')}
              </div>
              <div className='mt-1 flex flex-wrap gap-1.5'>
                {selectedAccount?.status ? (
                  <Tag
                    color={getWalletStatusMeta(selectedAccount.status, t).color}
                    size='small'
                  >
                    {getWalletStatusMeta(selectedAccount.status, t).label}
                  </Tag>
                ) : (
                  <Tag color='orange' size='small'>
                    {t('等待绑定')}
                  </Tag>
                )}
              </div>
            </div>
          ) : null}
        </div>

        {hasMessages ? (
          <div className='mt-2'>
            <button
              type='button'
              onClick={() => setWarningsExpanded(!warningsExpanded)}
              className='flex w-full items-center gap-2 rounded-lg px-3 py-1.5 text-left text-sm transition hover:bg-semi-color-fill-1'
            >
              {warningsExpanded ? (
                <ChevronDown size={14} className='text-semi-color-warning' />
              ) : (
                <ChevronRight size={14} className='text-semi-color-warning' />
              )}
              <AlertTriangle size={14} className='text-semi-color-warning' />
              <span className='text-semi-color-text-1'>
                {combinedWarnings.length > 0
                  ? t('{{count}} 个问题需要关注', {
                      count: combinedWarnings.length,
                    })
                  : ''}
                {combinedWarnings.length > 0 && sitePriceFactorNote ? '，' : ''}
                {sitePriceFactorNote ? t('有价格提示信息') : ''}
              </span>
              <span className='ml-auto text-xs text-semi-color-text-3'>
                {warningsExpanded ? t('收起') : t('展开')}
              </span>
            </button>
            <Collapsible isOpen={warningsExpanded}>
              <div className='mt-1 space-y-0.5 rounded-lg bg-semi-color-fill-0 px-3 py-2'>
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
          </div>
        ) : null}
      </Card>
    </>
  );
};

export default ProfitBoardHeader;
