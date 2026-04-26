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

import { useState, useEffect, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { Modal } from '@douyinfe/semi-ui';
import {
  API,
  getTodayStartTimestamp,
  isAdmin,
  showError,
  showSuccess,
  timestamp2string,
  renderQuota,
  renderNumber,
  getLogOther,
  copy,
  renderClaudeLogContent,
  renderLogContent,
  renderAudioModelPrice,
  renderClaudeModelPrice,
  renderModelPrice,
  renderTieredModelPrice,
} from '../../helpers';
import { ITEMS_PER_PAGE } from '../../constants';
import { useTableCompactMode } from '../common/useTableCompactMode';

export const useLogsData = () => {
  const { t } = useTranslation();

  // Define column keys for selection
  const COLUMN_KEYS = {
    TIME: 'time',
    CHANNEL: 'channel',
    USERNAME: 'username',
    TOKEN: 'token',
    GROUP: 'group',
    TYPE: 'type',
    MODEL: 'model',
    USE_TIME: 'use_time',
    PROMPT: 'prompt',
    COMPLETION: 'completion',
    COST: 'cost',
    RETRY: 'retry',
    IP: 'ip',
    DETAILS: 'details',
  };

  // Basic state
  const [logs, setLogs] = useState([]);
  const [expandData, setExpandData] = useState({});
  const [showStat, setShowStat] = useState(false);
  const [loading, setLoading] = useState(false);
  const [loadingStat, setLoadingStat] = useState(false);
  const [activePage, setActivePage] = useState(1);
  const [logCount, setLogCount] = useState(0);
  const [pageSize, setPageSize] = useState(ITEMS_PER_PAGE);
  const [logType, setLogType] = useState(0);
  const [groupOptions, setGroupOptions] = useState([]);
  const logsRequestCounter = useRef(0);
  const statRequestCounter = useRef(0);
  const topUsersRequestCounter = useRef(0);

  // User and admin
  const isAdminUser = isAdmin();
  // Role-specific storage key to prevent different roles from overwriting each other
  const STORAGE_KEY = isAdminUser
    ? 'logs-table-columns-admin'
    : 'logs-table-columns-user';
  const BILLING_DISPLAY_MODE_STORAGE_KEY = isAdminUser
    ? 'logs-billing-display-mode-admin'
    : 'logs-billing-display-mode-user';

  // Statistics state
  const [stat, setStat] = useState({
    quota: 0,
    token: 0,
  });

  // Form state
  const [formApi, setFormApi] = useState(null);
  let now = new Date();
  const formInitValues = {
    username: '',
    token_name: '',
    model_name: '',
    channel: '',
    group: '',
    request_id: '',
    dateRange: [
      timestamp2string(getTodayStartTimestamp()),
      timestamp2string(now.getTime() / 1000 + 3600),
    ],
    logType: '0',
  };

  // Get default column visibility based on user role
  const getDefaultColumnVisibility = () => {
    return {
      [COLUMN_KEYS.TIME]: true,
      [COLUMN_KEYS.CHANNEL]: isAdminUser,
      [COLUMN_KEYS.USERNAME]: isAdminUser,
      [COLUMN_KEYS.TOKEN]: true,
      [COLUMN_KEYS.GROUP]: true,
      [COLUMN_KEYS.TYPE]: true,
      [COLUMN_KEYS.MODEL]: true,
      [COLUMN_KEYS.USE_TIME]: true,
      [COLUMN_KEYS.PROMPT]: true,
      [COLUMN_KEYS.COMPLETION]: true,
      [COLUMN_KEYS.COST]: true,
      [COLUMN_KEYS.RETRY]: isAdminUser,
      [COLUMN_KEYS.IP]: true,
      [COLUMN_KEYS.DETAILS]: true,
    };
  };

  const getInitialVisibleColumns = () => {
    const defaults = getDefaultColumnVisibility();
    const savedColumns = localStorage.getItem(STORAGE_KEY);

    if (!savedColumns) {
      return defaults;
    }

    try {
      const parsed = JSON.parse(savedColumns);
      const merged = { ...defaults, ...parsed };

      if (!isAdminUser) {
        merged[COLUMN_KEYS.CHANNEL] = false;
        merged[COLUMN_KEYS.USERNAME] = false;
        merged[COLUMN_KEYS.RETRY] = false;
      }

      return merged;
    } catch (e) {
      console.error('Failed to parse saved column preferences', e);
      return defaults;
    }
  };

  const getInitialBillingDisplayMode = () => {
    const savedMode = localStorage.getItem(BILLING_DISPLAY_MODE_STORAGE_KEY);
    if (savedMode === 'price' || savedMode === 'ratio') {
      return savedMode;
    }
    return localStorage.getItem('quota_display_type') === 'TOKENS'
      ? 'ratio'
      : 'price';
  };

  // Column visibility state
  const [visibleColumns, setVisibleColumns] = useState(getInitialVisibleColumns);
  const [showColumnSelector, setShowColumnSelector] = useState(false);
  const [billingDisplayMode, setBillingDisplayMode] = useState(
    getInitialBillingDisplayMode,
  );

  // Compact mode
  const [compactMode, setCompactMode] = useTableCompactMode('logs');

  // User info modal state
  const [showUserInfo, setShowUserInfoModal] = useState(false);
  const [userInfoData, setUserInfoData] = useState(null);

  // Channel affinity usage cache stats modal state (admin only)
  const [
    showChannelAffinityUsageCacheModal,
    setShowChannelAffinityUsageCacheModal,
  ] = useState(false);
  const [channelAffinityUsageCacheTarget, setChannelAffinityUsageCacheTarget] =
    useState(null);

  // Top users drawer state (admin only)
  const [showTopUsersDrawer, setShowTopUsersDrawer] = useState(false);
  const [topUsersLoading, setTopUsersLoading] = useState(false);
  const [topUsersData, setTopUsersData] = useState({
    by_quota: [],
    by_requests: [],
  });
  const [topUsersViewMode, setTopUsersViewMode] = useState('both');
  const [topUsersQuotaOrder, setTopUsersQuotaOrder] = useState('desc');
  const [topUsersRequestOrder, setTopUsersRequestOrder] = useState('desc');
  const [topUsersLimit, setTopUsersLimit] = useState(10);

  // Initialize default column visibility
  const initDefaultColumns = () => {
    const defaults = getDefaultColumnVisibility();
    setVisibleColumns(defaults);
    localStorage.setItem(STORAGE_KEY, JSON.stringify(defaults));
  };

  // Handle column visibility change
  const handleColumnVisibilityChange = (columnKey, checked) => {
    const updatedColumns = { ...visibleColumns, [columnKey]: checked };
    setVisibleColumns(updatedColumns);
  };

  // Handle "Select All" checkbox
  const handleSelectAll = (checked) => {
    const allKeys = Object.keys(COLUMN_KEYS).map((key) => COLUMN_KEYS[key]);
    const updatedColumns = {};

    allKeys.forEach((key) => {
      if (
        (key === COLUMN_KEYS.CHANNEL ||
          key === COLUMN_KEYS.USERNAME ||
          key === COLUMN_KEYS.RETRY) &&
        !isAdminUser
      ) {
        updatedColumns[key] = false;
      } else {
        updatedColumns[key] = checked;
      }
    });

    setVisibleColumns(updatedColumns);
  };

  // Persist column settings to the role-specific STORAGE_KEY
  useEffect(() => {
    if (Object.keys(visibleColumns).length > 0) {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(visibleColumns));
    }
  }, [visibleColumns]);

  useEffect(() => {
    localStorage.setItem(BILLING_DISPLAY_MODE_STORAGE_KEY, billingDisplayMode);
  }, [BILLING_DISPLAY_MODE_STORAGE_KEY, billingDisplayMode]);

  // 获取表单值的辅助函数，确保所有值都是字符串
  const getFormValues = () => {
    const formValues = formApi ? formApi.getValues() : {};

    let start_timestamp = timestamp2string(getTodayStartTimestamp());
    let end_timestamp = timestamp2string(now.getTime() / 1000 + 3600);

    if (
      formValues.dateRange &&
      Array.isArray(formValues.dateRange) &&
      formValues.dateRange.length === 2
    ) {
      start_timestamp = formValues.dateRange[0];
      end_timestamp = formValues.dateRange[1];
    }

    return {
      username: formValues.username || '',
      token_name: formValues.token_name || '',
      model_name: formValues.model_name || '',
      start_timestamp,
      end_timestamp,
      channel: formValues.channel || '',
      group: formValues.group || '',
      request_id: formValues.request_id || '',
      logType: formValues.logType ? parseInt(formValues.logType) : 0,
    };
  };

  const toUnixTimestamp = (value) => {
    const parsed = Date.parse(value);
    return Number.isFinite(parsed) ? Math.floor(parsed / 1000) : 0;
  };

  const buildQueryString = (params) => {
    const query = new URLSearchParams();
    Object.entries(params).forEach(([key, value]) => {
      query.set(
        key,
        value === undefined || value === null ? '' : String(value),
      );
    });
    return query.toString();
  };

  const normalizeLogQueryValues = (customLogType = null) => {
    const values = getFormValues();
    const currentLogType =
      customLogType !== null
        ? customLogType
        : values.logType !== undefined
          ? values.logType
          : logType;

    return {
      ...values,
      logType: currentLogType,
      startTimestamp: toUnixTimestamp(values.start_timestamp),
      endTimestamp: toUnixTimestamp(values.end_timestamp),
    };
  };

  const normalizeGroupOptions = (data) => {
    if (Array.isArray(data)) {
      return data.map((group) => ({
        label: group,
        value: group,
      }));
    }

    if (data && typeof data === 'object') {
      return Object.entries(data).map(([group, info]) => ({
        label: info?.desc || group,
        value: group,
        ratio: info?.ratio,
      }));
    }

    return [];
  };

  const loadGroups = async () => {
    const url = isAdminUser ? '/api/group/' : '/api/user/self/groups';
    try {
      const res = await API.get(url);
      const { success, message, data } = res.data || {};
      if (success) {
        setGroupOptions(normalizeGroupOptions(data));
      } else {
        showError(t(message || '加载分组失败'));
      }
    } catch (error) {
      showError(error?.message || t('加载分组失败'));
    }
  };

  const loadTopUsers = async () => {
    if (!isAdminUser) {
      return;
    }
    const reqId = ++topUsersRequestCounter.current;
    const {
      username,
      token_name,
      model_name,
      startTimestamp,
      endTimestamp,
      channel,
      group,
      request_id,
    } = normalizeLogQueryValues();

    setTopUsersLoading(true);
    try {
      const params = new URLSearchParams({
        username,
        token_name,
        model_name,
        start_timestamp: String(startTimestamp),
        end_timestamp: String(endTimestamp),
        channel: String(channel),
        group,
        request_id,
        view_mode: topUsersViewMode,
        quota_order: topUsersQuotaOrder,
        request_order: topUsersRequestOrder,
        limit: String(topUsersLimit),
      });
      const res = await API.get(`/api/log/top-users?${params.toString()}`);
      if (reqId !== topUsersRequestCounter.current) {
        return;
      }
      const { success, message, data } = res.data || {};
      if (success) {
        setTopUsersData({
          by_quota: data?.by_quota || [],
          by_requests: data?.by_requests || [],
        });
      } else {
        showError(message || 'Failed to load top users');
      }
    } catch (error) {
      if (reqId !== topUsersRequestCounter.current) {
        return;
      }
      showError(error?.message || 'Failed to load top users');
    } finally {
      if (reqId === topUsersRequestCounter.current) {
        setTopUsersLoading(false);
      }
    }
  };

  const openTopUsersDrawer = () => {
    setShowTopUsersDrawer(true);
  };

  const selectTopUser = async (username) => {
    if (!formApi || !username) {
      return;
    }
    formApi.setValue('username', username);
    setShowTopUsersDrawer(false);
    await refresh();
  };

  // Statistics functions
  const getLogSelfStat = async (reqId) => {
    const {
      token_name,
      model_name,
      startTimestamp,
      endTimestamp,
      group,
      request_id,
      logType: currentLogType,
    } = normalizeLogQueryValues();
    const query = buildQueryString({
      type: currentLogType,
      token_name,
      model_name,
      start_timestamp: startTimestamp,
      end_timestamp: endTimestamp,
      group,
      request_id,
    });
    let url = `/api/log/self/stat?${query}`;
    let res = await API.get(url);
    if (reqId !== statRequestCounter.current) {
      return;
    }
    const { success, message, data } = res.data;
    if (success) {
      setStat(data);
    } else {
      showError(message);
    }
  };

  const getLogStat = async (reqId) => {
    const {
      username,
      token_name,
      model_name,
      startTimestamp,
      endTimestamp,
      channel,
      group,
      request_id,
      logType: currentLogType,
    } = normalizeLogQueryValues();
    const query = buildQueryString({
      type: currentLogType,
      username,
      token_name,
      model_name,
      start_timestamp: startTimestamp,
      end_timestamp: endTimestamp,
      channel,
      group,
      request_id,
    });
    let url = `/api/log/stat?${query}`;
    let res = await API.get(url);
    if (reqId !== statRequestCounter.current) {
      return;
    }
    const { success, message, data } = res.data;
    if (success) {
      setStat(data);
    } else {
      showError(message);
    }
  };

  const handleEyeClick = async () => {
    const reqId = ++statRequestCounter.current;
    setLoadingStat(true);
    try {
      if (isAdminUser) {
        await getLogStat(reqId);
      } else {
        await getLogSelfStat(reqId);
      }
      if (reqId === statRequestCounter.current) {
        setShowStat(true);
      }
    } catch (error) {
      if (reqId === statRequestCounter.current) {
        showError(error?.message || t('加载统计失败'));
      }
    } finally {
      if (reqId === statRequestCounter.current) {
        setLoadingStat(false);
      }
    }
  };

  // User info function
  const showUserInfoFunc = async (userId) => {
    if (!isAdminUser) {
      return;
    }
    const res = await API.get(`/api/user/${userId}`);
    const { success, message, data } = res.data;
    if (success) {
      setUserInfoData(data);
      setShowUserInfoModal(true);
    } else {
      showError(message);
    }
  };

  const openChannelAffinityUsageCacheModal = (affinity) => {
    const a = affinity || {};
    setChannelAffinityUsageCacheTarget({
      rule_name: a.rule_name || a.reason || '',
      using_group: a.using_group || '',
      key_hint: a.key_hint || '',
      key_fp: a.key_fp || '',
    });
    setShowChannelAffinityUsageCacheModal(true);
  };

  // Format logs data
  const setLogsFormat = (logs) => {
    const requestConversionDisplayValue = (conversionChain) => {
      const chain = Array.isArray(conversionChain)
        ? conversionChain.filter(Boolean)
        : [];
      if (chain.length <= 1) {
        return t('原生格式');
      }
      return `${chain.join(' -> ')}`;
    };

    let expandDatesLocal = {};
    for (let i = 0; i < logs.length; i++) {
      logs[i].timestamp2string = timestamp2string(logs[i].created_at);
      logs[i].key = logs[i].id;
      let other = getLogOther(logs[i].other);
      let expandDataLocal = [];

      if (isAdminUser && (logs[i].type === 0 || logs[i].type === 2 || logs[i].type === 6)) {
        expandDataLocal.push({
          key: t('渠道信息'),
          value: `${logs[i].channel} - ${logs[i].channel_name || '[未知]'}`,
        });
      }
      if (logs[i].request_id) {
        expandDataLocal.push({
          key: t('Request ID'),
          value: logs[i].request_id,
        });
      }
      if (other?.ws || other?.audio) {
        expandDataLocal.push({
          key: t('语音输入'),
          value: other.audio_input,
        });
        expandDataLocal.push({
          key: t('语音输出'),
          value: other.audio_output,
        });
        expandDataLocal.push({
          key: t('文字输入'),
          value: other.text_input,
        });
        expandDataLocal.push({
          key: t('文字输出'),
          value: other.text_output,
        });
      }
      if (other?.cache_tokens > 0) {
        expandDataLocal.push({
          key: t('缓存 Tokens'),
          value: other.cache_tokens,
        });
      }
      if (other?.cache_creation_tokens > 0) {
        expandDataLocal.push({
          key: t('缓存创建 Tokens'),
          value: other.cache_creation_tokens,
        });
      }
      if (logs[i].type === 2) {
        if (other?.billing_mode !== 'tiered_expr') {
          expandDataLocal.push({
            key: t('日志详情'),
            value: other?.claude
              ? renderClaudeLogContent({ ...other, displayMode: billingDisplayMode })
              : renderLogContent({ ...other, displayMode: billingDisplayMode }),
          });
        }
        if (logs[i]?.content) {
          expandDataLocal.push({
            key: t('其他详情'),
            value: logs[i].content,
          });
        }
        if (isAdminUser && other?.reject_reason) {
          expandDataLocal.push({
            key: t('拦截原因'),
            value: other.reject_reason,
          });
        }
      }
      if (logs[i].type === 2) {
        let modelMapped =
          other?.is_model_mapped &&
          other?.upstream_model_name &&
          other?.upstream_model_name !== '';
        if (modelMapped) {
          expandDataLocal.push({
            key: t('请求并计费模型'),
            value: logs[i].model_name,
          });
          expandDataLocal.push({
            key: t('实际模型'),
            value: other.upstream_model_name,
          });
        }

        const isViolationFeeLog =
          other?.violation_fee === true ||
          Boolean(other?.violation_fee_code) ||
          Boolean(other?.violation_fee_marker);

        let content = '';
        if (!isViolationFeeLog && other?.billing_mode !== 'tiered_expr') {
          const logOpts = {
            ...other,
            prompt_tokens: logs[i].prompt_tokens,
            completion_tokens: logs[i].completion_tokens,
            displayMode: billingDisplayMode,
          };
          if (other?.ws || other?.audio) {
            content = renderAudioModelPrice(logOpts);
          } else if (other?.claude) {
            content = renderClaudeModelPrice(logOpts);
          } else {
            content = renderModelPrice(logOpts);
          }
          expandDataLocal.push({
            key: t('计费过程'),
            value: content,
          });
        }
        if (other?.reasoning_effort) {
          expandDataLocal.push({
            key: t('Reasoning Effort'),
            value: other.reasoning_effort,
          });
        }
        if (other?.billing_mode === 'tiered_expr' && other?.expr_b64) {
          expandDataLocal.push({
            key: t('计费过程'),
            value: renderTieredModelPrice({
              ...other,
              prompt_tokens: logs[i].prompt_tokens,
              completion_tokens: logs[i].completion_tokens,
              displayMode: billingDisplayMode,
            }),
          });
        }
      }
      if (logs[i].type === 6) {
        if (other?.task_id) {
          expandDataLocal.push({
            key: t('任务ID'),
            value: other.task_id,
          });
        }
        if (other?.reason) {
          expandDataLocal.push({
            key: t('失败原因'),
            value: (
              <div style={{ maxWidth: 600, whiteSpace: 'normal', wordBreak: 'break-word', lineHeight: 1.6 }}>
                {other.reason}
              </div>
            ),
          });
        }
      }
      if (other?.request_path) {
        expandDataLocal.push({
          key: t('请求路径'),
          value: other.request_path,
        });
      }
      if (other?.billing_source === 'subscription') {
        const planId = other?.subscription_plan_id;
        const planTitle = other?.subscription_plan_title || '';
        const subscriptionId = other?.subscription_id;
        const unit = t('额度');
        const pre = other?.subscription_pre_consumed ?? 0;
        const postDelta = other?.subscription_post_delta ?? 0;
        const finalConsumed = other?.subscription_consumed ?? pre + postDelta;
        const remain = other?.subscription_remain;
        const total = other?.subscription_total;
        // Use multiple Description items to avoid an overlong single line.
        if (planId) {
          expandDataLocal.push({
            key: t('订阅套餐'),
            value: `#${planId} ${planTitle}`.trim(),
          });
        }
        if (subscriptionId) {
          expandDataLocal.push({
            key: t('订阅实例'),
            value: `#${subscriptionId}`,
          });
        }
        const settlementLines = [
          `${t('预扣')}：${pre} ${unit}`,
          `${t('结算差额')}：${postDelta > 0 ? '+' : ''}${postDelta} ${unit}`,
          `${t('最终抵扣')}：${finalConsumed} ${unit}`,
        ]
          .filter(Boolean)
          .join('\n');
        expandDataLocal.push({
          key: t('订阅结算'),
          value: (
            <div style={{ whiteSpace: 'pre-line' }}>{settlementLines}</div>
          ),
        });
        if (remain !== undefined && total !== undefined) {
          expandDataLocal.push({
            key: t('订阅剩余'),
            value: `${remain}/${total} ${unit}`,
          });
        }
        expandDataLocal.push({
          key: t('订阅说明'),
          value: t(
            'token 会按倍率换算成“额度/次数”，请求结束后再做差额结算（补扣/返还）。',
          ),
        });
      }
      if (isAdminUser && logs[i].type !== 6) {
        expandDataLocal.push({
          key: t('请求转换'),
          value: requestConversionDisplayValue(other?.request_conversion),
        });
      }
      if (isAdminUser && logs[i].type !== 6) {
        let localCountMode = '';
        if (other?.admin_info?.local_count_tokens) {
          localCountMode = t('本地计费');
        } else {
          localCountMode = t('上游返回');
        }
        expandDataLocal.push({
          key: t('计费模式'),
          value: localCountMode,
        });
      }
      expandDatesLocal[logs[i].key] = expandDataLocal;
    }

    setExpandData(expandDatesLocal);
    setLogs(logs);
  };

  // Load logs function
  const loadLogs = async (startIdx, pageSize, customLogType = null) => {
    const reqId = ++logsRequestCounter.current;
    setLoading(true);

    const {
      username,
      token_name,
      model_name,
      startTimestamp,
      endTimestamp,
      channel,
      group,
      request_id,
      logType: currentLogType,
    } = normalizeLogQueryValues(customLogType);

    const queryParams = {
      p: startIdx,
      page_size: pageSize,
      type: currentLogType,
      token_name,
      model_name,
      start_timestamp: startTimestamp,
      end_timestamp: endTimestamp,
      group,
      request_id,
    };
    if (isAdminUser) {
      queryParams.username = username;
      queryParams.channel = channel;
    }

    const url = `${
      isAdminUser ? '/api/log/' : '/api/log/self/'
    }?${buildQueryString(queryParams)}`;

    try {
      const res = await API.get(url);
      if (reqId !== logsRequestCounter.current) {
        return;
      }
      const { success, message, data } = res.data;
      if (success) {
        const newPageData = data.items;
        setActivePage(data.page);
        setPageSize(data.page_size);
        setLogCount(data.total);

        setLogsFormat(newPageData);
      } else {
        showError(message);
      }
    } catch (error) {
      if (reqId === logsRequestCounter.current) {
        showError(error?.message || t('加载日志失败'));
      }
    } finally {
      if (reqId === logsRequestCounter.current) {
        setLoading(false);
      }
    }
  };

  // Page handlers
  const handlePageChange = (page) => {
    setActivePage(page);
    loadLogs(page, pageSize).then((r) => {});
  };

  const handlePageSizeChange = async (size) => {
    localStorage.setItem('page-size', size + '');
    setPageSize(size);
    setActivePage(1);
    loadLogs(1, size)
      .then()
      .catch((reason) => {
        showError(reason);
      });
  };

  // Refresh function
  const refresh = async () => {
    setActivePage(1);
    handleEyeClick();
    await loadLogs(1, pageSize);
    if (showTopUsersDrawer) {
      await loadTopUsers();
    }
  };

  // Copy text function
  const copyText = async (e, text) => {
    e.stopPropagation();
    if (await copy(text)) {
      showSuccess('已复制：' + text);
    } else {
      Modal.error({ title: t('无法复制到剪贴板，请手动复制'), content: text });
    }
  };

  // Initialize data
  useEffect(() => {
    const localPageSize =
      parseInt(localStorage.getItem('page-size')) || ITEMS_PER_PAGE;
    setPageSize(localPageSize);
    loadLogs(activePage, localPageSize)
      .then()
      .catch((reason) => {
        showError(reason);
      });
    loadGroups().then();
  }, []);

  // Initialize statistics when formApi is available
  useEffect(() => {
    if (formApi) {
      handleEyeClick();
    }
  }, [formApi]);

  useEffect(() => {
    if (showTopUsersDrawer) {
      loadTopUsers().catch((reason) => {
        showError(reason?.message || reason || 'Failed to load top users');
      });
    }
  }, [
    showTopUsersDrawer,
    topUsersViewMode,
    topUsersQuotaOrder,
    topUsersRequestOrder,
    topUsersLimit,
  ]);

  // Check if any record has expandable content
  const hasExpandableRows = () => {
    return logs.some(
      (log) => expandData[log.key] && expandData[log.key].length > 0,
    );
  };

  return {
    // Basic state
    logs,
    expandData,
    showStat,
    loading,
    loadingStat,
    activePage,
    logCount,
    pageSize,
    logType,
    stat,
    isAdminUser,
    groupOptions,

    // Form state
    formApi,
    setFormApi,
    formInitValues,
    getFormValues,

    // Column visibility
    visibleColumns,
    showColumnSelector,
    setShowColumnSelector,
    billingDisplayMode,
    setBillingDisplayMode,
    handleColumnVisibilityChange,
    handleSelectAll,
    initDefaultColumns,
    COLUMN_KEYS,

    // Compact mode
    compactMode,
    setCompactMode,

    // User info modal
    showUserInfo,
    setShowUserInfoModal,
    userInfoData,
    showUserInfoFunc,

    // Channel affinity usage cache stats modal
    showChannelAffinityUsageCacheModal,
    setShowChannelAffinityUsageCacheModal,
    channelAffinityUsageCacheTarget,
    openChannelAffinityUsageCacheModal,

    // Top users drawer
    showTopUsersDrawer,
    setShowTopUsersDrawer,
    topUsersLoading,
    topUsersData,
    topUsersViewMode,
    setTopUsersViewMode,
    topUsersQuotaOrder,
    setTopUsersQuotaOrder,
    topUsersRequestOrder,
    setTopUsersRequestOrder,
    topUsersLimit,
    setTopUsersLimit,
    openTopUsersDrawer,
    selectTopUser,
    refreshTopUsers: loadTopUsers,
    currentTopUsersLogType: getFormValues().logType,

    // Functions
    loadLogs,
    handlePageChange,
    handlePageSizeChange,
    refresh,
    copyText,
    handleEyeClick,
    setLogsFormat,
    hasExpandableRows,
    setLogType,

    // Translation
    t,
  };
};
