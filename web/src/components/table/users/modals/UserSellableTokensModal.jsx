import React, { useEffect, useMemo, useState } from 'react';
import {
  Button,
  Empty,
  Input,
  Modal,
  Select,
  SideSheet,
  Space,
  Spin,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { IconPlusCircle } from '@douyinfe/semi-icons';
import {
  IllustrationNoResult,
  IllustrationNoResultDark,
} from '@douyinfe/semi-illustrations';
import { API, renderQuota, showError, showSuccess, timestamp2string } from '../../../../helpers';
import {
  formatConcurrencyLabel,
} from '../../../../helpers/render';
import { useIsMobile } from '../../../../hooks/common/useIsMobile';
import CardTable from '../../../common/ui/CardTable';

const { Text } = Typography;

const PERIOD_LABELS = {
  hourly: '每小时',
  daily: '每日',
  weekly: '每周',
  monthly: '每月',
  custom: '自定义',
};

const renderPeriodLabel = (t, period) => t(PERIOD_LABELS[period] || PERIOD_LABELS.custom);

const formatExpiryText = (t, expiredTime) => {
  if (Number(expiredTime || 0) === -1) {
    return t('长期有效');
  }
  return timestamp2string(expiredTime);
};

const mapIssueSourceLabel = (t, sourceType) => {
  if (sourceType === 'wallet') return t('钱包购买');
  if (sourceType === 'redeem') return t('兑换码');
  if (sourceType === 'admin') return t('管理员添加');
  return sourceType || '-';
};

const mapIssueModeLabel = (t, issueMode) => {
  if (issueMode === 'renew') return t('续费');
  if (issueMode === 'stack') return t('新建');
  return issueMode || '-';
};

const getTokenRuntimeState = (token) => {
  const nowUnix = Date.now() / 1000;
  const endTime = Number(token?.expired_time || 0);
  const status = Number(token?.status || 0);
  const remainQuota = Number(token?.remain_quota || 0);
  const isExpiredByTime = endTime > 0 && endTime !== -1 && endTime <= nowUnix;
  const isEnabled = status === 1;
  const isExhausted = status === 4 || (!isEnabled && remainQuota <= 0);
  return {
    isEnabled,
    canEnable: !isEnabled && !isExpiredByTime && !isExhausted,
    canDisable: isEnabled,
  };
};

const UserSellableTokensModal = ({
  visible,
  onCancel,
  user,
  t,
  onSuccess,
}) => {
  const isMobile = useIsMobile();
  const [loading, setLoading] = useState(false);
  const [productsLoading, setProductsLoading] = useState(false);
  const [contextLoading, setContextLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [singleActionLoading, setSingleActionLoading] = useState({
    id: 0,
    action: '',
  });
  const [batchLoading, setBatchLoading] = useState(false);

  const [tokens, setTokens] = useState([]);
  const [issuances, setIssuances] = useState([]);
  const [products, setProducts] = useState([]);
  const [productContext, setProductContext] = useState(null);

  const [selectedProductId, setSelectedProductId] = useState(0);
  const [issueMode, setIssueMode] = useState('stack');
  const [renewTargetTokenId, setRenewTargetTokenId] = useState(0);
  const [tokenName, setTokenName] = useState('');
  const [tokenGroup, setTokenGroup] = useState('');
  const [selectedTokenIds, setSelectedTokenIds] = useState([]);
  const [selectedIssuanceIds, setSelectedIssuanceIds] = useState([]);
  const [batchIssuanceCancelLoading, setBatchIssuanceCancelLoading] = useState(false);
  const [currentPage, setCurrentPage] = useState(1);
  const pageSize = 10;

  const [issuanceActionLoading, setIssuanceActionLoading] = useState({ id: 0, action: '' });

  // Admin confirm issuance modal state
  const [confirmIssuanceModalVisible, setConfirmIssuanceModalVisible] = useState(false);
  const [confirmIssuanceTarget, setConfirmIssuanceTarget] = useState(null);
  const [confirmIssuanceContextLoading, setConfirmIssuanceContextLoading] = useState(false);
  const [confirmIssuanceContext, setConfirmIssuanceContext] = useState(null);
  const [confirmIssueMode, setConfirmIssueMode] = useState('stack');
  const [confirmRenewTargetTokenId, setConfirmRenewTargetTokenId] = useState(0);
  const [confirmTokenName, setConfirmTokenName] = useState('');
  const [confirmTokenGroup, setConfirmTokenGroup] = useState('');
  const [confirmSubmitting, setConfirmSubmitting] = useState(false);

  const pendingCount = useMemo(() => {
    return issuances.filter((item) => item?.status === 'pending').length;
  }, [issuances]);

  const enabledProducts = useMemo(() => {
    return (products || []).filter((item) => Number(item?.status || 0) === 1);
  }, [products]);

  const productOptions = useMemo(() => {
    return enabledProducts.map((item) => ({
      label: `${item?.name || ''} · ${Number(item?.total_quota || 0) === 0 ? t('不限') : renderQuota(item.total_quota)}`,
      value: Number(item?.id || 0),
    }));
  }, [enabledProducts]);

  const productNameMap = useMemo(() => {
    const map = new Map();
    (products || []).forEach((item) => {
      map.set(Number(item?.id || 0), item?.name || `#${item?.id || '-'}`);
    });
    return map;
  }, [products]);

  const selectedProduct = productContext?.product || null;
  const groupOptions = productContext?.group_options || [];
  const renewableTargets = productContext?.renewable_targets || [];
  const issuanceByTokenId = useMemo(() => {
    const map = new Map();
    (issuances || []).forEach((item) => {
      const tokenId = Number(item?.token_id || 0);
      if (tokenId > 0) {
        map.set(tokenId, item);
      }
    });
    return map;
  }, [issuances]);
  const pagedTokens = useMemo(() => {
    const start = Math.max(0, (Number(currentPage || 1) - 1) * pageSize);
    return (tokens || []).slice(start, start + pageSize);
  }, [tokens, currentPage]);
  const actionLabelMap = useMemo(
    () => ({
      enable: t('启用'),
      disable: t('禁用'),
      delete: t('删除'),
    }),
    [t],
  );

  const renewTargetOptions = useMemo(() => {
    return renewableTargets.map((token) => ({
      value: Number(token?.id || 0),
      label: `${token?.name || '-'} (#${token?.id || '-'}) · ${t('到期')} ${formatExpiryText(
        t,
        token?.expired_time,
      )}`,
    }));
  }, [renewableTargets, t]);

  const canRenew = renewTargetOptions.length > 0;

  const issueModeOptions = useMemo(() => {
    return [
      { value: 'stack', label: t('叠加新令牌') },
      canRenew
        ? { value: 'renew', label: t('续费已有令牌') }
        : {
            value: 'renew',
            label: `${t('续费已有令牌')} (${t('暂无可续费目标')})`,
            disabled: true,
          },
    ];
  }, [canRenew, t]);

  const tokenColumns = useMemo(() => {
    return [
      { title: 'ID', dataIndex: 'id', width: 72 },
      {
        title: t('商品'),
        width: 220,
        render: (_, record) => {
          const issuance = issuanceByTokenId.get(Number(record?.id || 0));
          const sourceLabel = mapIssueSourceLabel(t, issuance?.source_type);
          return (
            <div className='min-w-0'>
              <div className='font-medium truncate'>
                {productNameMap.get(Number(record?.product_id || 0)) ||
                  `#${record?.product_id || '-'}`}
              </div>
              <div className='text-xs text-gray-500'>
                {t('来源')}：{sourceLabel}
              </div>
            </div>
          );
        },
      },
      {
        title: t('结束时间'),
        width: 180,
        render: (_, record) => formatExpiryText(t, record?.expired_time),
      },
      {
        title: t('额度'),
        width: 180,
        render: (_, record) =>
          `${renderQuota(record?.remain_quota || 0)} / ${renderQuota(
            Number(record?.remain_quota || 0) + Number(record?.used_quota || 0),
          )}`,
      },
      {
        title: t('状态'),
        dataIndex: 'status',
        width: 100,
        render: (text) => (
          <Tag color={Number(text) === 1 ? 'green' : 'grey'} shape='circle'>
            {Number(text) === 1
              ? t('启用')
              : Number(text) === 2
              ? t('禁用')
              : Number(text) === 3
              ? t('已过期')
              : Number(text) === 4
              ? t('已耗尽')
              : text}
          </Tag>
        ),
      },
      {
        title: '',
        key: 'operate',
        width: 230,
        fixed: 'right',
        render: (_, record) => {
          const state = getTokenRuntimeState(record);
          const loadingId = Number(singleActionLoading.id || 0);
          const loadingAction = singleActionLoading.action;
          const currentTokenId = Number(record?.id || 0);
          const busy = batchLoading || submitting || loading;
          return (
            <Space>
              <Button
                size='small'
                theme='light'
                type='tertiary'
                disabled={!state.canEnable || busy}
                loading={loadingId === currentTokenId && loadingAction === 'enable'}
                onClick={() => confirmManageToken(currentTokenId, 'enable')}
              >
                {t('启用')}
              </Button>
              <Button
                size='small'
                theme='light'
                type='warning'
                disabled={!state.canDisable || busy}
                loading={loadingId === currentTokenId && loadingAction === 'disable'}
                onClick={() => confirmManageToken(currentTokenId, 'disable')}
              >
                {t('禁用')}
              </Button>
              <Button
                size='small'
                theme='light'
                type='danger'
                disabled={busy}
                loading={loadingId === currentTokenId && loadingAction === 'delete'}
                onClick={() => confirmManageToken(currentTokenId, 'delete')}
              >
                {t('删除')}
              </Button>
            </Space>
          );
        },
      },
    ];
  }, [
    productNameMap,
    issuanceByTokenId,
    singleActionLoading,
    batchLoading,
    submitting,
    loading,
    t,
  ]);

  const issuanceColumns = useMemo(() => {
    return [
      { title: 'ID', dataIndex: 'id', width: 72 },
      {
        title: t('商品'),
        width: 220,
        render: (_, record) => {
          const productName = record?.product?.name || '-';
          const sourceLabel = mapIssueSourceLabel(t, record?.source_type);
          return (
            <div className='min-w-0'>
              <div className='font-medium truncate'>{productName}</div>
              <div className='text-xs text-gray-500'>
                {t('来源')}：{sourceLabel}
              </div>
            </div>
          );
        },
      },
      {
        title: t('状态'),
        dataIndex: 'status',
        width: 100,
        render: (text) => (
          <Tag color={text === 'pending' ? 'orange' : text === 'cancelled' ? 'red' : 'green'} shape='circle'>
            {text === 'pending' ? t('待发放') : text === 'cancelled' ? t('已取消') : t('已发放')}
          </Tag>
        ),
      },
      {
        title: t('发放方式'),
        dataIndex: 'issue_mode',
        width: 120,
        render: (text) => mapIssueModeLabel(t, text),
      },
      {
        title: t('发放时间'),
        dataIndex: 'issued_time',
        width: 180,
        render: (text) => (Number(text || 0) > 0 ? timestamp2string(text) : '-'),
      },
      {
        title: '',
        key: 'operate',
        width: 200,
        fixed: 'right',
        render: (_, record) => {
          if (record?.status !== 'pending') return null;
          const recordId = Number(record?.id || 0);
          const busy = loading || submitting || issuanceActionLoading.id > 0;
          return (
            <Space>
              <Button
                size='small'
                theme='solid'
                type='primary'
                disabled={busy}
                loading={issuanceActionLoading.id === recordId && issuanceActionLoading.action === 'confirm'}
                onClick={() => openConfirmIssuanceModal(record)}
              >
                {t('发放')}
              </Button>
              <Button
                size='small'
                theme='light'
                type='danger'
                disabled={busy}
                loading={issuanceActionLoading.id === recordId && issuanceActionLoading.action === 'cancel'}
                onClick={() => confirmCancelIssuance(recordId)}
              >
                {t('取消')}
              </Button>
            </Space>
          );
        },
      },
    ];
  }, [t, loading, submitting, issuanceActionLoading]);

  const rowSelection = useMemo(() => {
    return {
      selectedRowKeys: selectedTokenIds,
      onChange: (selectedRowKeys) => {
        setSelectedTokenIds(
          (selectedRowKeys || [])
            .map((key) => Number(key || 0))
            .filter((id) => id > 0),
        );
      },
    };
  }, [selectedTokenIds]);

  const loadSummary = async () => {
    if (!user?.id) return;
    setLoading(true);
    try {
      const res = await API.get(`/api/user/${user.id}/sellable-token/summary`);
      const { success, message, data } = res.data || {};
      if (success) {
        setTokens(data?.tokens || []);
        setIssuances(data?.issuances || []);
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message || t('加载用户可售令牌失败'));
    } finally {
      setLoading(false);
    }
  };

  const loadProducts = async () => {
    setProductsLoading(true);
    try {
      const res = await API.get('/api/redemption/sellable-token-products');
      const { success, message, data } = res.data || {};
      if (success) {
        setProducts(data || []);
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error.message || t('加载可售令牌商品失败'));
    } finally {
      setProductsLoading(false);
    }
  };

  const loadProductContext = async (productId) => {
    if (!user?.id || Number(productId || 0) <= 0) {
      setProductContext(null);
      return;
    }
    setContextLoading(true);
    try {
      const res = await API.get(
        `/api/user/${user.id}/sellable-token/products/${productId}/context`,
      );
      const { success, message, data } = res.data || {};
      if (success) {
        setProductContext(data || null);
      } else {
        setProductContext(null);
        showError(message);
      }
    } catch (error) {
      setProductContext(null);
      showError(error.message || t('加载可售令牌商品上下文失败'));
    } finally {
      setContextLoading(false);
    }
  };

  useEffect(() => {
    if (!visible) {
      setTokens([]);
      setIssuances([]);
      setProducts([]);
      setProductContext(null);
      setSelectedProductId(0);
      setIssueMode('stack');
      setRenewTargetTokenId(0);
      setTokenName('');
      setTokenGroup('');
      setSelectedTokenIds([]);
      setSelectedIssuanceIds([]);
      setCurrentPage(1);
      return;
    }
    loadSummary();
    loadProducts();
  }, [visible, user?.id]);

  useEffect(() => {
    if (!visible) return;
    if (Number(selectedProductId || 0) <= 0) {
      setProductContext(null);
      setIssueMode('stack');
      setRenewTargetTokenId(0);
      setTokenName('');
      setTokenGroup('');
      return;
    }
    loadProductContext(selectedProductId);
  }, [visible, selectedProductId, user?.id]);

  useEffect(() => {
    if (!selectedProduct) return;
    const nextGroup =
      productContext?.requested_group || groupOptions?.[0]?.value || '';
    const firstTarget = Number(renewTargetOptions?.[0]?.value || 0);
    setIssueMode('stack');
    setRenewTargetTokenId(firstTarget);
    setTokenName(selectedProduct?.name || '');
    setTokenGroup(nextGroup);
  }, [selectedProduct, productContext?.requested_group, groupOptions, renewTargetOptions]);

  useEffect(() => {
    if (issueMode !== 'renew') {
      setRenewTargetTokenId(0);
      return;
    }
    if (renewTargetOptions.length === 1) {
      setRenewTargetTokenId(Number(renewTargetOptions[0].value || 0));
      return;
    }
    const exists = renewTargetOptions.some(
      (option) => Number(option?.value || 0) === Number(renewTargetTokenId || 0),
    );
    if (!exists) {
      setRenewTargetTokenId(0);
    }
  }, [issueMode, renewTargetOptions, renewTargetTokenId]);

  useEffect(() => {
    const exists = new Set(
      (tokens || []).map((item) => Number(item?.id || 0)).filter((id) => id > 0),
    );
    setSelectedTokenIds((prev) => prev.filter((id) => exists.has(id)));
  }, [tokens]);

  useEffect(() => {
    const pendingIds = new Set(
      (issuances || [])
        .filter((item) => item?.status === 'pending')
        .map((item) => Number(item?.id || 0))
        .filter((id) => id > 0),
    );
    setSelectedIssuanceIds((prev) => prev.filter((id) => pendingIds.has(id)));
  }, [issuances]);

  const submitIssue = async () => {
    if (!user?.id) {
      showError(t('用户信息缺失'));
      return;
    }
    if (Number(selectedProductId || 0) <= 0) {
      showError(t('请选择可售令牌商品'));
      return;
    }
    if (!tokenGroup) {
      showError(t('请选择分组'));
      return;
    }
    if (issueMode === 'renew' && !canRenew) {
      showError(t('当前无可续费令牌'));
      return;
    }
    if (
      issueMode === 'renew' &&
      renewTargetOptions.length > 1 &&
      Number(renewTargetTokenId || 0) <= 0
    ) {
      showError(t('请选择续费目标'));
      return;
    }

    setSubmitting(true);
    try {
      const res = await API.post(`/api/user/${user.id}/sellable-token/issue`, {
        product_id: Number(selectedProductId),
        mode: issueMode,
        target_token_id:
          issueMode === 'renew' ? Number(renewTargetTokenId || 0) : 0,
        name: tokenName || '',
        group: tokenGroup || '',
      });
      const { success, message } = res.data || {};
      if (success) {
        showSuccess(t('可售令牌发放成功'));
        await Promise.all([loadSummary(), loadProductContext(selectedProductId)]);
        onSuccess?.();
      } else {
        showError(message || t('发放可售令牌失败'));
      }
    } catch (error) {
      showError(error.message || t('发放可售令牌失败'));
    } finally {
      setSubmitting(false);
    }
  };

  const confirmIssue = () => {
    Modal.confirm({
      title: t('确认操作'),
      content:
        issueMode === 'renew'
          ? t('是否确认按当前设置续费可售令牌？')
          : t('是否确认新增可售令牌？'),
      centered: true,
      onOk: async () => {
        await submitIssue();
      },
    });
  };

  const handlePageChange = (page) => {
    setCurrentPage(page);
  };

  const manageToken = async (tokenId, action) => {
    if (!user?.id || !tokenId || !action) return;
    setSingleActionLoading({ id: Number(tokenId), action });
    try {
      const res = await API.post(
        `/api/user/${user.id}/sellable-token/tokens/manage`,
        { id: Number(tokenId), action },
      );
      if (res.data?.success) {
        const msg =
          res.data?.data?.message ||
          t('操作成功：{{action}}', { action: actionLabelMap[action] || action });
        showSuccess(msg);
        await loadSummary();
        onSuccess?.();
      } else {
        showError(res.data?.message || t('操作失败'));
      }
    } catch (error) {
      showError(error.message || t('请求失败'));
    } finally {
      setSingleActionLoading({ id: 0, action: '' });
    }
  };

  const confirmManageToken = (tokenId, action) => {
    const isDelete = action === 'delete';
    const title =
      action === 'enable'
        ? t('确认启用')
        : action === 'disable'
        ? t('确认禁用')
        : t('确认删除');
    const content =
      action === 'enable'
        ? t('仅未过期且未耗尽的令牌可启用。是否继续？')
        : action === 'disable'
        ? t('禁用后不会删除令牌，可稍后重新启用。是否继续？')
        : t('删除会彻底移除该令牌记录。是否继续？');
    Modal.confirm({
      title,
      content,
      centered: true,
      okType: isDelete ? 'danger' : 'primary',
      onOk: async () => {
        await manageToken(tokenId, action);
      },
    });
  };

  const batchManageTokens = async (action) => {
    if (!user?.id || selectedTokenIds.length === 0) return;
    setBatchLoading(true);
    try {
      const res = await API.post(
        `/api/user/${user.id}/sellable-token/tokens/manage/batch`,
        {
          ids: selectedTokenIds,
          action,
        },
      );
      if (res.data?.success) {
        const result = res.data?.data || {};
        const successCount = Number(result?.success_count || 0);
        const failedCount = Number(result?.failed_count || 0);
        if (failedCount > 0) {
          const firstFailedMessage = result?.failed?.[0]?.message;
          showError(
            t('批量{{action}}完成：成功 {{success}} 条，失败 {{failed}} 条', {
              action: actionLabelMap[action] || action,
              success: successCount,
              failed: failedCount,
            }) + (firstFailedMessage ? `；${firstFailedMessage}` : ''),
          );
        } else {
          showSuccess(
            t('批量{{action}}成功：{{count}} 条', {
              action: actionLabelMap[action] || action,
              count: successCount,
            }),
          );
        }
        setSelectedTokenIds([]);
        await loadSummary();
        onSuccess?.();
      } else {
        showError(res.data?.message || t('批量操作失败'));
      }
    } catch (error) {
      showError(error.message || t('请求失败'));
    } finally {
      setBatchLoading(false);
    }
  };

  const confirmBatchDelete = () => {
    if (selectedTokenIds.length === 0) return;
    Modal.confirm({
      title: t('确认批量删除'),
      content: t('确定要删除所选的 {{count}} 条令牌吗？', {
        count: selectedTokenIds.length,
      }),
      centered: true,
      okType: 'danger',
      onOk: async () => {
        await batchManageTokens('delete');
      },
    });
  };

  const batchCancelIssuances = async () => {
    if (!user?.id || selectedIssuanceIds.length === 0) return;
    setBatchIssuanceCancelLoading(true);
    try {
      const res = await API.post(
        `/api/user/${user.id}/sellable-token/issuances/cancel/batch`,
        { ids: selectedIssuanceIds },
      );
      if (res.data?.success) {
        const result = res.data?.data || {};
        const successCount = Number(result?.success_count || 0);
        const failedCount = Number(result?.failed_count || 0);
        if (failedCount > 0) {
          const firstFailedMessage = result?.failed?.[0]?.message;
          showError(
            t('批量取消完成：成功 {{success}} 条，失败 {{failed}} 条', {
              success: successCount,
              failed: failedCount,
            }) + (firstFailedMessage ? `；${firstFailedMessage}` : ''),
          );
        } else {
          showSuccess(
            t('批量取消成功：{{count}} 条', { count: successCount }),
          );
        }
        setSelectedIssuanceIds([]);
        await loadSummary();
        onSuccess?.();
      } else {
        showError(res.data?.message || t('批量取消失败'));
      }
    } catch (error) {
      showError(error.message || t('请求失败'));
    } finally {
      setBatchIssuanceCancelLoading(false);
    }
  };

  const confirmBatchCancelIssuances = () => {
    if (selectedIssuanceIds.length === 0) return;
    Modal.confirm({
      title: t('确认批量取消'),
      content: t('确定要取消所选的 {{count}} 条待发放记录吗？钱包购买的将退还额度。', {
        count: selectedIssuanceIds.length,
      }),
      centered: true,
      okType: 'danger',
      okText: t('确认取消'),
      onOk: async () => {
        await batchCancelIssuances();
      },
    });
  };

  const confirmCancelIssuance = (issuanceId) => {
    Modal.confirm({
      title: t('确认取消'),
      content: t('取消后不可恢复，钱包购买的将退还额度。是否继续？'),
      centered: true,
      okType: 'danger',
      okText: t('确认取消'),
      onOk: async () => {
        setIssuanceActionLoading({ id: issuanceId, action: 'cancel' });
        try {
          const res = await API.post(
            `/api/user/${user.id}/sellable-token/issuances/${issuanceId}/cancel`,
          );
          if (res.data?.success) {
            showSuccess(t('已取消'));
            await loadSummary();
            onSuccess?.();
          } else {
            showError(res.data?.message || t('取消失败'));
          }
        } catch (error) {
          showError(error.message || t('请求失败'));
        } finally {
          setIssuanceActionLoading({ id: 0, action: '' });
        }
      },
    });
  };

  const openConfirmIssuanceModal = async (record) => {
    const productId = Number(record?.product?.id || 0);
    setConfirmIssuanceTarget(record);
    setConfirmIssuanceModalVisible(true);
    setConfirmIssueMode('stack');
    setConfirmRenewTargetTokenId(0);
    setConfirmTokenName(record?.product?.name || '');
    setConfirmTokenGroup('');
    if (productId > 0) {
      setConfirmIssuanceContextLoading(true);
      try {
        const res = await API.get(
          `/api/user/${user.id}/sellable-token/products/${productId}/context`,
        );
        if (res.data?.success) {
          const ctx = res.data.data || null;
          setConfirmIssuanceContext(ctx);
          setConfirmTokenGroup(ctx?.group_options?.[0]?.value || '');
        } else {
          setConfirmIssuanceContext(null);
          showError(res.data?.message || t('加载上下文失败'));
        }
      } catch (e) {
        setConfirmIssuanceContext(null);
      } finally {
        setConfirmIssuanceContextLoading(false);
      }
    }
  };

  const submitConfirmIssuance = async () => {
    if (!confirmIssuanceTarget || !user?.id) return;
    const issuanceId = Number(confirmIssuanceTarget?.id || 0);
    if (issuanceId <= 0) return;
    if (!confirmTokenGroup) {
      showError(t('请选择分组'));
      return;
    }
    setConfirmSubmitting(true);
    try {
      const res = await API.post(
        `/api/user/${user.id}/sellable-token/issuances/${issuanceId}/confirm`,
        {
          mode: confirmIssueMode,
          target_token_id: confirmIssueMode === 'renew' ? Number(confirmRenewTargetTokenId || 0) : 0,
          name: confirmTokenName || '',
          group: confirmTokenGroup || '',
        },
      );
      if (res.data?.success) {
        showSuccess(t('可售令牌发放成功'));
        setConfirmIssuanceModalVisible(false);
        setConfirmIssuanceTarget(null);
        setConfirmIssuanceContext(null);
        await loadSummary();
        onSuccess?.();
      } else {
        showError(res.data?.message || t('发放失败'));
      }
    } catch (error) {
      showError(error.message || t('请求失败'));
    } finally {
      setConfirmSubmitting(false);
    }
  };

  const confirmIssuanceGroupOptions = confirmIssuanceContext?.group_options || [];
  const confirmIssuanceRenewableTargets = confirmIssuanceContext?.renewable_targets || [];
  const confirmIssuanceCanRenew = confirmIssuanceRenewableTargets.length > 0;
  const confirmIssuanceRenewOptions = confirmIssuanceRenewableTargets.map((token) => ({
    value: Number(token?.id || 0),
    label: `${token?.name || '-'} (#${token?.id || '-'}) · ${t('到期')} ${formatExpiryText(t, token?.expired_time)}`,
  }));

  return (
    <SideSheet
      visible={visible}
      placement='right'
      width={isMobile ? '100%' : 980}
      bodyStyle={{ padding: 0 }}
      onCancel={onCancel}
      title={
        <Space>
          <Tag color='blue' shape='circle'>
            {t('管理')}
          </Tag>
          <Typography.Title heading={4} className='m-0'>
            {t('用户令牌管理')}
          </Typography.Title>
          <Text type='tertiary' className='ml-2'>
            {user?.username || '-'} (ID: {user?.id || '-'})
          </Text>
        </Space>
      }
    >
      <Spin spinning={loading}>
        <div className='p-4'>
          <div className='mb-4 rounded-xl border border-solid border-[var(--semi-color-border)] bg-[var(--semi-color-fill-0)] p-4'>
            <div className='flex flex-col gap-3'>
              <div className='flex flex-col lg:flex-row gap-2'>
                <Select
                  placeholder={t('选择可售令牌商品')}
                  optionList={productOptions}
                  value={selectedProductId || undefined}
                  onChange={(value) => setSelectedProductId(Number(value || 0))}
                  loading={productsLoading}
                  filter
                  style={{ minWidth: isMobile ? undefined : 320, flex: 1 }}
                />
                <Select
                  placeholder={t('发放方式')}
                  optionList={issueModeOptions}
                  value={issueMode}
                  onChange={(value) => setIssueMode(value || 'stack')}
                  disabled={Number(selectedProductId || 0) <= 0 || contextLoading}
                  style={{ minWidth: isMobile ? undefined : 160 }}
                />
                <Button
                  type='primary'
                  theme='solid'
                  icon={<IconPlusCircle />}
                  loading={submitting}
                  disabled={
                    Number(selectedProductId || 0) <= 0 ||
                    contextLoading ||
                    !tokenGroup ||
                    (issueMode === 'renew' && !canRenew)
                  }
                  onClick={confirmIssue}
                >
                  {t('新增令牌')}
                </Button>
              </div>

              {selectedProduct ? (
                <>
                  <div className='grid grid-cols-1 gap-2 lg:grid-cols-[minmax(0,1fr)_220px]'>
                    <Input
                      value={tokenName}
                      placeholder={t('请输入令牌名称')}
                      onChange={setTokenName}
                      maxLength={50}
                      disabled={contextLoading}
                    />
                    <Select
                      placeholder={t('选择分组')}
                      optionList={groupOptions.map((item) => ({
                        label: item?.label || item?.value,
                        value: item?.value,
                      }))}
                      value={tokenGroup || undefined}
                      onChange={(value) => setTokenGroup(value || '')}
                      disabled={contextLoading}
                      style={{ minWidth: isMobile ? undefined : 220, width: '100%' }}
                    />
                  </div>

                  {issueMode === 'renew' && renewTargetOptions.length > 1 ? (
                    <Select
                      placeholder={t('选择续费目标')}
                      optionList={renewTargetOptions}
                      value={renewTargetTokenId || undefined}
                      onChange={(value) => setRenewTargetTokenId(Number(value || 0))}
                      disabled={contextLoading}
                      style={{ minWidth: isMobile ? undefined : 460 }}
                    />
                  ) : null}

                  {issueMode === 'renew' && renewTargetOptions.length === 1 ? (
                    <Text type='tertiary'>
                      {t('续费目标')}: {renewTargetOptions[0].label}
                    </Text>
                  ) : null}

                  <Space wrap spacing={8}>
                    <Tag color='white' shape='circle'>
                      {t('总额度')} {Number(selectedProduct?.total_quota || 0) === 0 ? t('不限') : renderQuota(selectedProduct.total_quota)}
                    </Tag>
                    <Tag color='white' shape='circle'>
                      {t('有效期')}{' '}
                      {Number(selectedProduct?.validity_seconds || 0) > 0
                        ? `${selectedProduct?.validity_seconds || 0}s`
                        : t('长期有效')}
                    </Tag>
                    {selectedProduct?.package_enabled ? (
                      <Tag color='white' shape='circle'>
                        {t('周期额度上限')} {renderQuota(selectedProduct?.package_limit_quota || 0)} /{' '}
                        {renderPeriodLabel(t, selectedProduct?.package_period)}
                      </Tag>
                    ) : null}
                    <Tag color='white' shape='circle'>
                      {Number(selectedProduct?.max_concurrency || 0) > 0
                        ? formatConcurrencyLabel(selectedProduct.max_concurrency, t)
                        : `${t('并发')} ${t('不限')}`}
                    </Tag>
                  </Space>
                </>
              ) : (
                <Text type='tertiary'>{t('请选择要发放的可售令牌商品')}</Text>
              )}
            </div>
          </div>

          <div className='mb-4 flex flex-wrap gap-2'>
            <Tag color='green' shape='circle'>
              {t('已发放')} {tokens.length}
            </Tag>
            <Tag color='blue' shape='circle'>
              {t('待发放')} {pendingCount}
            </Tag>
            <Tag color='white' shape='circle'>
              {t('总记录')} {issuances.length}
            </Tag>
          </div>

          <div className='flex flex-col md:flex-row md:items-center md:justify-between gap-2 mb-3'>
            <Text type='tertiary'>
              {t('已选择 {{count}} 条令牌', {
                count: selectedTokenIds.length,
              })}
            </Text>
            <Space wrap>
              <Button
                size='small'
                type='tertiary'
                disabled={selectedTokenIds.length === 0 || batchLoading}
                loading={batchLoading}
                onClick={() => batchManageTokens('enable')}
              >
                {t('批量启用')}
              </Button>
              <Button
                size='small'
                type='tertiary'
                disabled={selectedTokenIds.length === 0 || batchLoading}
                loading={batchLoading}
                onClick={() => batchManageTokens('disable')}
              >
                {t('批量禁用')}
              </Button>
              <Button
                size='small'
                type='danger'
                disabled={selectedTokenIds.length === 0 || batchLoading}
                loading={batchLoading}
                onClick={confirmBatchDelete}
              >
                {t('批量删除')}
              </Button>
            </Space>
          </div>

          <CardTable
            columns={tokenColumns}
            dataSource={pagedTokens}
            rowKey={(row) => Number(row?.id || 0)}
            rowSelection={!isMobile ? rowSelection : undefined}
            loading={loading}
            scroll={{ x: 'max-content' }}
            hidePagination={false}
            pagination={{
              currentPage,
              pageSize,
              total: tokens.length,
              pageSizeOpts: [10, 20, 50],
              showSizeChanger: false,
              onPageChange: handlePageChange,
            }}
            empty={
              <Empty
                image={
                  <IllustrationNoResult style={{ width: 150, height: 150 }} />
                }
                darkModeImage={
                  <IllustrationNoResultDark
                    style={{ width: 150, height: 150 }}
                  />
                }
                description={t('暂无令牌记录')}
                style={{ padding: 30 }}
              />
            }
          />

          <div className='mt-6'>
            <div className='mb-3 flex flex-col items-start gap-1'>
              <Typography.Title heading={6} className='!mb-0'>
                {t('令牌待发放记录')}
              </Typography.Title>
              <Text type='tertiary' size='small'>
                {t('用于查看用户已购买、已兑换或管理员手动发放的可售令牌记录')}
              </Text>
            </div>
            <div className='flex flex-col md:flex-row md:items-center md:justify-between gap-2 mb-3'>
              <Text type='tertiary'>
                {t('已选择 {{count}} 条待发放', { count: selectedIssuanceIds.length })}
              </Text>
              <Button
                size='small'
                type='danger'
                disabled={selectedIssuanceIds.length === 0 || batchIssuanceCancelLoading}
                loading={batchIssuanceCancelLoading}
                onClick={confirmBatchCancelIssuances}
              >
                {t('批量取消')}
              </Button>
            </div>
            <CardTable
              columns={issuanceColumns}
              dataSource={issuances}
              rowKey={(row) => Number(row?.id || 0)}
              rowSelection={!isMobile ? {
                selectedRowKeys: selectedIssuanceIds,
                onChange: (selectedRowKeys) => {
                  setSelectedIssuanceIds(
                    (selectedRowKeys || [])
                      .map((key) => Number(key || 0))
                      .filter((id) => id > 0),
                  );
                },
                getCheckboxProps: (record) => ({
                  disabled: record?.status !== 'pending',
                }),
              } : undefined}
              pagination={false}
              scroll={{ x: 960 }}
              empty={<Empty description={t('暂无待发放记录')} />}
            />
          </div>
        </div>
      </Spin>

      <Modal
        visible={confirmIssuanceModalVisible}
        title={t('代用户发放可售令牌')}
        centered
        maskClosable={false}
        onCancel={() => {
          setConfirmIssuanceModalVisible(false);
          setConfirmIssuanceTarget(null);
          setConfirmIssuanceContext(null);
        }}
        onOk={submitConfirmIssuance}
        confirmLoading={confirmSubmitting}
        okText={t('确认发放')}
        width={520}
      >
        <Spin spinning={confirmIssuanceContextLoading}>
          <div className='space-y-3'>
            {confirmIssuanceTarget?.product?.name && (
              <div>
                <Text type='tertiary' size='small'>{t('商品')}</Text>
                <div className='font-medium'>{confirmIssuanceTarget.product.name}</div>
              </div>
            )}
            <div>
              <Text type='tertiary' size='small'>{t('发放方式')}</Text>
              <Select
                optionList={[
                  { value: 'stack', label: t('叠加新令牌') },
                  confirmIssuanceCanRenew
                    ? { value: 'renew', label: t('续费已有令牌') }
                    : { value: 'renew', label: `${t('续费已有令牌')} (${t('暂无可续费目标')})`, disabled: true },
                ]}
                value={confirmIssueMode}
                onChange={(v) => setConfirmIssueMode(v || 'stack')}
                style={{ width: '100%' }}
              />
            </div>
            <div>
              <Text type='tertiary' size='small'>{t('令牌名称')}</Text>
              <Input
                value={confirmTokenName}
                onChange={setConfirmTokenName}
                placeholder={t('请输入令牌名称')}
                maxLength={50}
              />
            </div>
            <div>
              <Text type='tertiary' size='small'>{t('分组')}</Text>
              <Select
                optionList={confirmIssuanceGroupOptions.map((item) => ({
                  label: item?.label || item?.value,
                  value: item?.value,
                }))}
                value={confirmTokenGroup || undefined}
                onChange={(v) => setConfirmTokenGroup(v || '')}
                placeholder={t('选择分组')}
                style={{ width: '100%' }}
              />
            </div>
            {confirmIssueMode === 'renew' && confirmIssuanceRenewOptions.length > 1 && (
              <div>
                <Text type='tertiary' size='small'>{t('续费目标')}</Text>
                <Select
                  optionList={confirmIssuanceRenewOptions}
                  value={confirmRenewTargetTokenId || undefined}
                  onChange={(v) => setConfirmRenewTargetTokenId(Number(v || 0))}
                  placeholder={t('选择续费目标')}
                  style={{ width: '100%' }}
                />
              </div>
            )}
            {confirmIssueMode === 'renew' && confirmIssuanceRenewOptions.length === 1 && (
              <Text type='tertiary'>
                {t('续费目标')}: {confirmIssuanceRenewOptions[0].label}
              </Text>
            )}
          </div>
        </Spin>
      </Modal>
    </SideSheet>
  );
};

export default UserSellableTokensModal;
