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
} from 'lucide-react';
import { groupWarningDetailsByScope } from '../utils';

const { Text, Title } = Typography;
const WARNING_DETAIL_DEFAULT_LIMIT = 10;

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
  combinedMessages,
  hasUnsavedConfigChanges,
  configReady,
  t,
}) => {
  const [warningsExpanded, setWarningsExpanded] = useState(false);
  const [expandedWarningKeys, setExpandedWarningKeys] = useState({});
  const [expandedWarningMore, setExpandedWarningMore] = useState({});
  const allMessages = combinedMessages || [];
  const hasMessages = allMessages.length > 0;

  const toggleWarningDetails = (key) => {
    setExpandedWarningKeys((current) => ({
      ...current,
      [key]: !current[key],
    }));
  };

  const toggleWarningMore = (key) => {
    setExpandedWarningMore((current) => ({
      ...current,
      [key]: !current[key],
    }));
  };

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
          {!configReady && (
            <Tag color='blue' size='small'>
              {t('正在从服务器加载配置...')}
            </Tag>
          )}
          {hasMessages && (
            <Tag
              color='amber'
              size='small'
              className='cursor-pointer'
              onClick={() => setWarningsExpanded(!warningsExpanded)}
            >
              {allMessages.length} {t('条提示')}
            </Tag>
          )}
        </div>
        <Space wrap>
          {hasUnsavedConfigChanges && (
            <Tag color='orange' size='small'>
              {t('当前有未保存更改，仅保存在当前浏览器')}
            </Tag>
          )}
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

      {statusSummary?.length > 0 ? (
        <div className='mt-2 flex flex-wrap gap-2'>
          {statusSummary.map((item) => (
            <Tag key={item.key} color={item.color} size='small'>
              {item.text}
            </Tag>
          ))}
        </div>
      ) : null}

      {hasMessages && (
        <Collapsible isOpen={warningsExpanded}>
          <div className='mt-2 space-y-0.5 rounded-lg bg-semi-color-fill-0 px-3 py-2'>
            {allMessages.map((msg, idx) => {
              const messageKey = msg.key || `${msg.type}-${idx}-${msg.text}`;
              const detailItems = Array.isArray(msg.details) ? msg.details : [];
              const groupedDetails =
                msg.type === 'warning' ? groupWarningDetailsByScope(detailItems) : [];
              const hasDetails = msg.type === 'warning' && groupedDetails.length > 0;
              const detailExpanded = !!expandedWarningKeys[messageKey];
              const showAllDetails = !!expandedWarningMore[messageKey];
              const overflowDetailCount = Math.max(
                0,
                groupedDetails.length - WARNING_DETAIL_DEFAULT_LIMIT,
              );
              const visibleDetails = showAllDetails
                ? groupedDetails
                : groupedDetails.slice(0, WARNING_DETAIL_DEFAULT_LIMIT);
              const hiddenDetailCount = Math.max(
                0,
                groupedDetails.length - visibleDetails.length,
              );

              return (
                <div
                  key={messageKey}
                  className='rounded-md py-1'
                >
                  <div className='flex items-start justify-between gap-3 text-sm'>
                    <div className='flex min-w-0 items-start gap-2'>
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
                      <div className='min-w-0'>
                        <div className='break-words text-semi-color-text-1'>
                          {msg.text}
                        </div>
                        {hasDetails && (
                          <div className='mt-1 flex flex-wrap items-center gap-2'>
                            <Text type='tertiary' size='small'>
                              {t('共 {{count}} 条未命中', {
                                count: msg.totalCount || detailItems.length,
                              })}
                            </Text>
                            <Button
                              theme='borderless'
                              type='primary'
                              size='small'
                              icon={
                                detailExpanded ? (
                                  <ChevronDown size={14} />
                                ) : (
                                  <ChevronRight size={14} />
                                )
                              }
                              onClick={() => toggleWarningDetails(messageKey)}
                            >
                              {detailExpanded ? t('收起明细') : t('查看明细')}
                            </Button>
                          </div>
                        )}
                      </div>
                    </div>
                  </div>

                  {hasDetails && (
                    <Collapsible isOpen={detailExpanded}>
                      <div className='mt-2 space-y-2 pl-5'>
                        {visibleDetails.map((detail, detailIdx) => (
                          <div
                            key={`${messageKey}-${detail.scopeKey || detailIdx}`}
                            className='rounded-md border border-semi-color-border bg-white px-3 py-3'
                          >
                            <div className='flex items-start justify-between gap-3'>
                              <div className='min-w-0 space-y-2'>
                                <div className='flex flex-wrap items-center gap-2'>
                                  <Tag
                                    color={detail.scopeType === 'tag' ? 'blue' : 'grey'}
                                    size='small'
                                  >
                                    {detail.scopeType === 'tag'
                                      ? t('标签')
                                      : t('渠道')}
                                  </Tag>
                                  <Text strong className='break-all'>
                                    {detail.scopeLabel}
                                  </Text>
                                  {detail.displayHint ? (
                                    <Text type='tertiary' size='small'>
                                      {detail.displayHint}
                                    </Text>
                                  ) : null}
                                </div>
                                <div className='space-y-1.5'>
                                  {detail.models.map((model) => (
                                    <div
                                      key={`${detail.scopeKey}-${model.modelName || 'model'}`}
                                      className='flex items-center justify-between gap-3 rounded-md bg-semi-color-fill-0 px-2.5 py-1.5'
                                    >
                                      <div className='flex min-w-0 items-center gap-2'>
                                        <Tag color='grey' size='small'>
                                          {t('模型')}
                                        </Tag>
                                        <Text type='tertiary' size='small' className='break-all'>
                                          {model.modelName}
                                        </Text>
                                      </div>
                                      <Text type='tertiary' size='small'>
                                        {t('{{count}} 次', { count: model.count })}
                                      </Text>
                                    </div>
                                  ))}
                                </div>
                              </div>
                              <div className='shrink-0 text-right'>
                                <Tag
                                  color='orange'
                                  shape='circle'
                                  size='small'
                                >
                                  {t('未命中')}
                                </Tag>
                                <div className='mt-1'>
                                  <Text strong>{detail.totalCount}</Text>
                                </div>
                                <Text type='tertiary' size='small'>
                                  {t('次未命中')}
                                </Text>
                              </div>
                            </div>
                          </div>
                        ))}
                        {overflowDetailCount > 0 && (
                          <Button
                            theme='borderless'
                            type='primary'
                            size='small'
                            onClick={() => toggleWarningMore(messageKey)}
                          >
                            {showAllDetails
                              ? t('收起多余项')
                              : t('再展开 {{count}} 项', {
                                  count: hiddenDetailCount,
                                })}
                          </Button>
                        )}
                      </div>
                    </Collapsible>
                  )}
                </div>
              );
            })}
          </div>
        </Collapsible>
      )}
    </>
  );
};

export default ProfitBoardHeader;
