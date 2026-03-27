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

import React, { useEffect, useState, useContext, useRef, useMemo } from 'react';
import {
  API,
  showError,
  showInfo,
  showSuccess,
  timestamp2string,
  renderGroupOption,
  getModelCategories,
  selectFilter,
  isAdmin,
} from '../../../../helpers';
import { quotaToUSDAmount, usdAmountToQuota } from '../../../../helpers/quota';
import { useIsMobile } from '../../../../hooks/common/useIsMobile';
import {
  Button,
  Checkbox,
  SideSheet,
  Select,
  Space,
  Spin,
  Switch,
  Typography,
  Card,
  Tag,
  Avatar,
  Form,
  Col,
  Row,
} from '@douyinfe/semi-ui';
import {
  IconCreditCard,
  IconLink,
  IconSave,
  IconClose,
  IconKey,
} from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import { StatusContext } from '../../../../context/Status';

const { Text, Title } = Typography;
const TOKEN_CHANNEL_LIMIT_TAG_MODE_KEY = 'token-channel-limit-tag-mode';

const EditTokenModal = (props) => {
  const { t } = useTranslation();
  const [statusState, statusDispatch] = useContext(StatusContext);
  const [loading, setLoading] = useState(false);
  const isMobile = useIsMobile();
  const formApiRef = useRef(null);
  const loadedTokenValuesRef = useRef(null);
  const [models, setModels] = useState([]);
  const [groups, setGroups] = useState([]);
  const [channelOptions, setChannelOptions] = useState([]);
  const [tokenMode, setTokenMode] = useState('standard');
  const isAdminUser = isAdmin();
  const isEdit = props.editingToken.id !== undefined;
  const isSellableToken = props.editingToken?.source_type === 'sellable_token';
  const channelRequestRef = useRef(0);
  const [channelLimitTagMode, setChannelLimitTagMode] = useState(() => {
    if (typeof window === 'undefined') {
      return false;
    }
    return localStorage.getItem(TOKEN_CHANNEL_LIMIT_TAG_MODE_KEY) === 'true';
  });

  const getInitValues = () => ({
    name: '',
    remain_quota: 0,
    remain_amount: 0,
    expired_time: -1,
    unlimited_quota: true,
    model_limits_enabled: false,
    model_limits: [],
    channel_limits_enabled: false,
    channel_limits: [],
    allow_ips: '',
    group: '',
    cross_group_retry: false,
    max_concurrency: 0,
    window_request_limit: 0,
    window_seconds: 0,
    tokenCount: 1,
    package_enabled: false,
    package_limit_amount: 0,
    package_limit_quota: 0,
    package_period: 'daily',
    package_period_mode: 'relative',
    package_custom_seconds: 86400,
    package_used_quota: 0,
    package_next_reset_time: 0,
  });

  const applyLoadedTokenValues = () => {
    if (!formApiRef.current) return;
    const values = loadedTokenValuesRef.current;
    if (!values) return;
    formApiRef.current.setValues(values);
    if (values.package_limit_amount !== undefined) {
      // 首次挂载时显式回填套餐额度，避免初始 0.00 残留。
      formApiRef.current.setValue(
        'package_limit_amount',
        values.package_limit_amount,
      );
    }
  };

  const handleCancel = () => {
    props.handleClose();
  };

  const setExpiredTime = (month, day, hour, minute) => {
    let now = new Date();
    let timestamp = now.getTime() / 1000;
    let seconds = month * 30 * 24 * 60 * 60;
    seconds += day * 24 * 60 * 60;
    seconds += hour * 60 * 60;
    seconds += minute * 60;
    if (!formApiRef.current) return;
    if (seconds !== 0) {
      timestamp += seconds;
      formApiRef.current.setValue('expired_time', timestamp2string(timestamp));
    } else {
      formApiRef.current.setValue('expired_time', -1);
    }
  };

  const buildModelOptions = (data) => {
    const categories = getModelCategories(t);
    return data.map((model) => {
      let icon = null;
      for (const [key, category] of Object.entries(categories)) {
        if (key !== 'all' && category.filter({ model_name: model })) {
          icon = category.icon;
          break;
        }
      }
      return {
        label: (
          <span className='flex items-center gap-1'>
            {icon}
            {model}
          </span>
        ),
        value: model,
      };
    });
  };

  const syncModelLimitsWithOptions = (optionList, shouldNotify = false) => {
    if (!formApiRef.current) return;
    const allowedValues = new Set(optionList.map((item) => item.value));
    const currentValues = formApiRef.current.getValue('model_limits') || [];
    const nextValues = currentValues.filter((model) =>
      allowedValues.has(model),
    );
    if (nextValues.length === currentValues.length) return;
    formApiRef.current.setValue('model_limits', nextValues);
    if (loadedTokenValuesRef.current) {
      loadedTokenValuesRef.current = {
        ...loadedTokenValuesRef.current,
        model_limits: nextValues,
      };
    }
    if (shouldNotify) {
      showInfo(t('已自动移除当前分组不可用的模型限制'));
    }
  };

  const buildChannelOptions = (data) => {
    return (Array.isArray(data) ? data : []).map((channel) => {
      const channelId = String(channel.id);
      const channelName = `${channel.name} (#${channel.id})`;
      const channelTag =
        typeof channel.tag === 'string' ? channel.tag.trim() : '';
      const matchedGroupsArray = Array.isArray(channel.matched_groups)
        ? channel.matched_groups
        : [];
      const matchedModelsArray = Array.isArray(channel.matched_models)
        ? channel.matched_models
        : [];
      const matchedGroups = Array.isArray(channel.matched_groups)
        ? channel.matched_groups.join(', ')
        : '';
      const matchedModels = Array.isArray(channel.matched_models)
        ? channel.matched_models.slice(0, 3).join(', ')
        : '';
      const hiddenModelCount = Array.isArray(channel.matched_models)
        ? Math.max(channel.matched_models.length - 3, 0)
        : 0;
      const summaryParts = [];
      if (matchedGroups) {
        summaryParts.push(`${t('分组')}: ${matchedGroups}`);
      }
      if (matchedModels) {
        summaryParts.push(
          `${t('模型')}: ${matchedModels}${hiddenModelCount > 0 ? ` +${hiddenModelCount}` : ''}`,
        );
      }
      const summary = summaryParts.join(' | ');
      return {
        label: (
          <div className='flex flex-col'>
            <span>{channelName}</span>
            {summary ? (
              <span className='text-xs text-gray-500'>{summary}</span>
            ) : null}
          </div>
        ),
        value: channelId,
        channelName,
        channelSummary: summary,
        channelTag,
        matchedGroups: matchedGroupsArray,
        matchedModels: matchedModelsArray,
      };
    });
  };

  const alignChannelLimitValues = (
    channelLimits,
    optionList = channelOptions,
  ) => {
    const allowedValues = optionList.map((item) => String(item.value));
    const selectedSet = new Set(
      (channelLimits || []).map((channelId) => String(channelId)),
    );
    return allowedValues.filter((channelId) => selectedSet.has(channelId));
  };

  const applyChannelLimitValues = (
    channelLimits,
    optionList = channelOptions,
  ) => {
    if (!formApiRef.current) return;
    const nextValues = alignChannelLimitValues(channelLimits, optionList);
    formApiRef.current.setValue('channel_limits', nextValues);
    if (loadedTokenValuesRef.current) {
      loadedTokenValuesRef.current = {
        ...loadedTokenValuesRef.current,
        channel_limits: nextValues,
      };
    }
  };

  const syncChannelLimitsWithOptions = (optionList, shouldNotify = false) => {
    if (!formApiRef.current) return;
    const currentValues = formApiRef.current.getValue('channel_limits') || [];
    const nextValues = alignChannelLimitValues(currentValues, optionList);
    if (nextValues.length === currentValues.length) return;
    applyChannelLimitValues(nextValues, optionList);
    if (shouldNotify) {
      showInfo(t('已自动移除当前分组或模型限制下不可用的渠道'));
    }
  };

  const loadModels = async (groupValue = '', shouldNotify = false) => {
    let res = await API.get(`/api/token/models`, {
      params: {
        group: groupValue,
      },
    });
    const { success, message, data } = res.data;
    if (success) {
      const localModelOptions = buildModelOptions(
        Array.isArray(data) ? data : [],
      );
      setModels(localModelOptions);
      syncModelLimitsWithOptions(localModelOptions, shouldNotify);
    } else {
      setModels([]);
      showError(t(message));
    }
  };

  const loadChannels = async (
    groupValue = '',
    modelLimits = [],
    shouldNotify = false,
  ) => {
    if (!isAdminUser || isSellableToken) {
      setChannelOptions([]);
      return;
    }
    const requestId = ++channelRequestRef.current;
    let res = await API.get(`/api/token/channels`, {
      params: {
        group: groupValue,
        model_limits: Array.isArray(modelLimits)
          ? modelLimits.join(',')
          : String(modelLimits || ''),
        token_id: isEdit ? props.editingToken.id : undefined,
      },
    });
    if (requestId !== channelRequestRef.current) return;
    const { success, message, data } = res.data;
    if (success) {
      const localChannelOptions = buildChannelOptions(data);
      setChannelOptions(localChannelOptions);
      syncChannelLimitsWithOptions(localChannelOptions, shouldNotify);
    } else {
      setChannelOptions([]);
      showError(t(message));
    }
  };

  const loadGroups = async () => {
    let res = await API.get(`/api/user/self/groups`);
    const { success, message, data } = res.data;
    if (success) {
      let localGroupOptions = Object.entries(data).map(([group, info]) => ({
        label: info.desc,
        value: group,
        ratio: info.ratio,
      }));
      if (statusState?.status?.default_use_auto_group) {
        if (localGroupOptions.some((group) => group.value === 'auto')) {
          localGroupOptions.sort((a, b) => (a.value === 'auto' ? -1 : 1));
        }
      }
      // For sellable tokens, restrict groups to those allowed by the product
      if (isSellableToken && props.editingToken?.sellable_token_product_id) {
        try {
          const productsRes = await API.get(
            '/api/user/sellable-token/products',
          );
          if (productsRes.data?.success) {
            const products = productsRes.data.data || [];
            const matchedProduct = products.find(
              (item) =>
                Number(item?.product?.id || 0) ===
                Number(props.editingToken.sellable_token_product_id),
            );
            if (matchedProduct) {
              const allowedGroups = matchedProduct.allowed_groups || [];
              const userGroups = (matchedProduct.user_groups || []).map(
                (g) => g?.value,
              );
              if (allowedGroups.length > 0 || userGroups.length > 0) {
                const allowedSet = new Set(
                  userGroups.length > 0 ? userGroups : allowedGroups,
                );
                localGroupOptions = localGroupOptions.filter((g) =>
                  allowedSet.has(g.value),
                );
              }
            }
          }
        } catch (_) {}
      }
      setGroups(localGroupOptions);
    } else {
      showError(t(message));
    }
  };

  const loadToken = async () => {
    loadedTokenValuesRef.current = null;
    setLoading(true);
    let res = await API.get(`/api/token/${props.editingToken.id}`);
    const { success, message, data } = res.data;
    if (success) {
      if (data.expired_time !== -1) {
        data.expired_time = timestamp2string(data.expired_time);
      }
      if (data.model_limits !== '') {
        data.model_limits = data.model_limits.split(',');
      } else {
        data.model_limits = [];
      }
      if (data.channel_limits !== '') {
        data.channel_limits = data.channel_limits.split(',');
      } else {
        data.channel_limits = [];
      }
      if (!data.package_period || data.package_period === 'none') {
        data.package_period = 'daily';
      }
      data.remain_amount = quotaToUSDAmount(data.remain_quota || 0);
      data.package_limit_amount = quotaToUSDAmount(
        data.package_limit_quota || 0,
      );
      loadedTokenValuesRef.current = { ...getInitValues(), ...data };
      applyLoadedTokenValues();
      await loadModels(data.group || '');
      await loadChannels(data.group || '', data.model_limits || []);
      setTokenMode(data.package_enabled ? 'package' : 'standard');
    } else {
      showError(message);
    }
    setLoading(false);
  };

  useEffect(() => {
    loadedTokenValuesRef.current = null;
    setChannelOptions([]);
    if (formApiRef.current) {
      if (!isEdit) {
        formApiRef.current.setValues(getInitValues());
      }
    }
    loadModels(props.editingToken.group || '');
    loadChannels(
      props.editingToken.group || '',
      props.editingToken.channel_limits || [],
    );
    loadGroups();
  }, [props.editingToken.id]);

  useEffect(() => {
    if (props.visiable) {
      if (isEdit) {
        loadToken();
      } else {
        loadedTokenValuesRef.current = null;
        formApiRef.current?.setValues(getInitValues());
        setTokenMode('standard');
        loadModels('');
        loadChannels('', []);
      }
    } else {
      loadedTokenValuesRef.current = null;
      formApiRef.current?.reset();
      setChannelOptions([]);
    }
  }, [props.visiable, props.editingToken.id]);

  const channelTagGroups = useMemo(() => {
    const tagGroupMap = new Map();
    const untaggedChannels = [];

    channelOptions.forEach((option) => {
      const channelTag = (option.channelTag || '').trim();
      if (!channelTag) {
        untaggedChannels.push(option);
        return;
      }

      if (!tagGroupMap.has(channelTag)) {
        tagGroupMap.set(channelTag, {
          tag: channelTag,
          value: `tag:${channelTag}`,
          channelIds: [],
          channels: [],
          matchedGroupsSet: new Set(),
          matchedModelsSet: new Set(),
        });
      }

      const tagGroup = tagGroupMap.get(channelTag);
      tagGroup.channelIds.push(option.value);
      tagGroup.channels.push(option);
      (option.matchedGroups || []).forEach((groupName) =>
        tagGroup.matchedGroupsSet.add(groupName),
      );
      (option.matchedModels || []).forEach((modelName) =>
        tagGroup.matchedModelsSet.add(modelName),
      );
    });

    return {
      tagGroups: Array.from(tagGroupMap.values())
        .map((tagGroup) => ({
          ...tagGroup,
          matchedGroups: Array.from(tagGroup.matchedGroupsSet).sort(),
          matchedModels: Array.from(tagGroup.matchedModelsSet).sort(),
        }))
        .sort((a, b) => a.tag.localeCompare(b.tag, 'zh-CN')),
      untaggedChannels,
    };
  }, [channelOptions]);

  const updateChannelLimitTagMode = (enabled) => {
    localStorage.setItem(
      TOKEN_CHANNEL_LIMIT_TAG_MODE_KEY,
      enabled ? 'true' : 'false',
    );
    setChannelLimitTagMode(enabled);
  };

  const getChannelTagSelectionCount = (selectedSet, channelIds) => {
    return channelIds.reduce(
      (count, channelId) =>
        count + (selectedSet.has(String(channelId)) ? 1 : 0),
      0,
    );
  };

  const getChannelTagSummary = (tagGroup) => {
    const summaryParts = [`${tagGroup.channelIds.length}${t(' 个渠道')}`];
    if (tagGroup.matchedGroups.length > 0) {
      summaryParts.push(`${t('分组')}: ${tagGroup.matchedGroups.join(', ')}`);
    }
    if (tagGroup.matchedModels.length > 0) {
      const visibleModels = tagGroup.matchedModels.slice(0, 3).join(', ');
      const hiddenModelCount = Math.max(tagGroup.matchedModels.length - 3, 0);
      summaryParts.push(
        `${t('模型')}: ${visibleModels}${hiddenModelCount > 0 ? ` +${hiddenModelCount}` : ''}`,
      );
    }
    return summaryParts.join(' | ');
  };

  const toggleChannelTagGroup = (tagGroup, checked) => {
    const currentValues = formApiRef.current?.getValue('channel_limits') || [];
    const selectedSet = new Set(
      currentValues.map((channelId) => String(channelId)),
    );
    tagGroup.channelIds.forEach((channelId) => {
      if (checked) {
        selectedSet.add(String(channelId));
      } else {
        selectedSet.delete(String(channelId));
      }
    });
    applyChannelLimitValues(Array.from(selectedSet));
  };

  const toggleSingleChannelLimit = (channelId, checked) => {
    const currentValues = formApiRef.current?.getValue('channel_limits') || [];
    const selectedSet = new Set(currentValues.map((value) => String(value)));
    if (checked) {
      selectedSet.add(String(channelId));
    } else {
      selectedSet.delete(String(channelId));
    }
    applyChannelLimitValues(Array.from(selectedSet));
  };

  const removeChannelLimitValues = (channelIds) => {
    const currentValues = formApiRef.current?.getValue('channel_limits') || [];
    const removingSet = new Set(
      (channelIds || []).map((channelId) => String(channelId)),
    );
    const nextValues = currentValues.filter(
      (channelId) => !removingSet.has(String(channelId)),
    );
    applyChannelLimitValues(nextValues);
  };

  const switchTokenMode = (mode) => {
    setTokenMode(mode);
    if (!formApiRef.current) return;
    const isPackage = mode === 'package';
    formApiRef.current.setValue('package_enabled', isPackage);
    if (isPackage) {
      const currentPeriod = formApiRef.current.getValue('package_period');
      if (!currentPeriod || currentPeriod === 'none') {
        formApiRef.current.setValue('package_period', 'daily');
      }
      const currentLimit = Number(
        formApiRef.current.getValue('package_limit_quota') || 0,
      );
      if (currentLimit <= 0) {
        formApiRef.current.setValue('package_limit_amount', 10);
        formApiRef.current.setValue(
          'package_limit_quota',
          usdAmountToQuota(10),
        );
      }
    } else {
      formApiRef.current.setValue('package_limit_amount', 0);
      formApiRef.current.setValue('package_period', 'daily');
      formApiRef.current.setValue('package_limit_quota', 0);
      formApiRef.current.setValue('package_custom_seconds', 86400);
      formApiRef.current.setValue('package_used_quota', 0);
      formApiRef.current.setValue('package_next_reset_time', 0);
    }
  };

  const normalizePackageFields = (localInputs) => {
    const isPackage = tokenMode === 'package';
    localInputs.package_enabled = isPackage;
    if (!isPackage) {
      localInputs.package_limit_amount = 0;
      localInputs.package_limit_quota = 0;
      localInputs.package_period = 'none';
      localInputs.package_period_mode = 'relative';
      localInputs.package_custom_seconds = 0;
      localInputs.package_used_quota = 0;
      localInputs.package_next_reset_time = 0;
      return { ok: true };
    }
    localInputs.package_limit_amount = Number(
      localInputs.package_limit_amount || 0,
    );
    localInputs.package_limit_quota = usdAmountToQuota(
      localInputs.package_limit_amount,
    );
    localInputs.package_used_quota =
      parseInt(localInputs.package_used_quota, 10) || 0;
    localInputs.package_next_reset_time =
      parseInt(localInputs.package_next_reset_time, 10) || 0;
    const period = (localInputs.package_period || '').trim();
    if (!['hourly', 'daily', 'weekly', 'monthly', 'custom'].includes(period)) {
      return { ok: false, message: t('套餐周期无效') };
    }
    localInputs.package_period = period;
    // custom 周期本身就是相对的，强制设为 relative
    if (period === 'custom') {
      localInputs.package_period_mode = 'relative';
    } else {
      const mode = (localInputs.package_period_mode || '').trim();
      localInputs.package_period_mode = ['relative', 'natural'].includes(mode)
        ? mode
        : 'relative';
    }
    if (localInputs.package_limit_quota <= 0) {
      return { ok: false, message: t('周期金额必须大于 0') };
    }
    if (period === 'custom') {
      localInputs.package_custom_seconds =
        parseInt(localInputs.package_custom_seconds, 10) || 0;
      if (localInputs.package_custom_seconds <= 0) {
        return { ok: false, message: t('自定义周期秒数必须大于 0') };
      }
    } else {
      localInputs.package_custom_seconds = 0;
    }
    if (localInputs.package_used_quota < 0) {
      localInputs.package_used_quota = 0;
    }
    if (localInputs.package_next_reset_time < 0) {
      localInputs.package_next_reset_time = 0;
    }
    return { ok: true };
  };

  const normalizeRemainQuotaFields = (localInputs) => {
    localInputs.remain_amount = Number(localInputs.remain_amount || 0);
    if (localInputs.unlimited_quota) {
      localInputs.remain_quota = parseInt(localInputs.remain_quota, 10) || 0;
      return { ok: true };
    }
    localInputs.remain_quota = usdAmountToQuota(localInputs.remain_amount);
    if (localInputs.remain_quota < 0) {
      localInputs.remain_quota = 0;
    }
    return { ok: true };
  };

  const normalizeRuntimeLimitFields = (localInputs) => {
    localInputs.max_concurrency =
      parseInt(localInputs.max_concurrency, 10) || 0;
    localInputs.window_request_limit =
      parseInt(localInputs.window_request_limit, 10) || 0;
    localInputs.window_seconds = parseInt(localInputs.window_seconds, 10) || 0;

    if (localInputs.max_concurrency < 0) {
      return { ok: false, message: t('并发上限不能小于 0') };
    }
    if (localInputs.window_request_limit < 0) {
      return { ok: false, message: t('窗口请求上限不能小于 0') };
    }
    if (localInputs.window_seconds < 0) {
      return { ok: false, message: t('窗口时长不能小于 0') };
    }
    if (
      localInputs.window_request_limit > 0 &&
      localInputs.window_seconds <= 0
    ) {
      return {
        ok: false,
        message: t('设置请求窗口限制时，窗口时长必须大于 0'),
      };
    }
    if (
      localInputs.window_seconds > 0 &&
      localInputs.window_request_limit <= 0
    ) {
      return {
        ok: false,
        message: t('设置窗口时长时，请同时设置窗口请求上限'),
      };
    }
    return { ok: true };
  };

  const generateRandomSuffix = () => {
    const characters =
      'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
    let result = '';
    for (let i = 0; i < 6; i++) {
      result += characters.charAt(
        Math.floor(Math.random() * characters.length),
      );
    }
    return result;
  };

  const sanitizeModelLimits = (modelLimits) => {
    const allowedValues = new Set(models.map((item) => item.value));
    return (modelLimits || []).filter((model) => allowedValues.has(model));
  };

  const sanitizeChannelLimits = (channelLimits) => {
    return alignChannelLimitValues(channelLimits);
  };

  const submit = async (values) => {
    setLoading(true);
    if (isEdit) {
      let { tokenCount: _tc, ...localInputs } = values;
      if (!isSellableToken) {
        const remainResult = normalizeRemainQuotaFields(localInputs);
        if (!remainResult.ok) {
          showError(remainResult.message);
          setLoading(false);
          return;
        }
        const packageResult = normalizePackageFields(localInputs);
        if (!packageResult.ok) {
          showError(packageResult.message);
          setLoading(false);
          return;
        }
        const runtimeLimitResult = normalizeRuntimeLimitFields(localInputs);
        if (!runtimeLimitResult.ok) {
          showError(runtimeLimitResult.message);
          setLoading(false);
          return;
        }
        localInputs.model_limits = sanitizeModelLimits(
          localInputs.model_limits,
        );
        localInputs.channel_limits = isAdminUser
          ? sanitizeChannelLimits(localInputs.channel_limits)
          : [];
        delete localInputs.remain_amount;
        delete localInputs.package_limit_amount;
        if (localInputs.expired_time !== -1) {
          let time = Date.parse(localInputs.expired_time);
          if (isNaN(time)) {
            showError(t('过期时间格式错误！'));
            setLoading(false);
            return;
          }
          localInputs.expired_time = Math.ceil(time / 1000);
        }
        localInputs.model_limits = localInputs.model_limits.join(',');
        localInputs.model_limits_enabled = localInputs.model_limits.length > 0;
        localInputs.channel_limits = localInputs.channel_limits.join(',');
        localInputs.channel_limits_enabled =
          localInputs.channel_limits.length > 0;
      } else {
        localInputs = {
          name: localInputs.name,
          group: localInputs.group,
          status: props.editingToken.status,
        };
      }
      let res = await API.put(`/api/token/`, {
        ...localInputs,
        id: parseInt(props.editingToken.id),
      });
      const { success, message } = res.data;
      if (success) {
        showSuccess(t('令牌更新成功！'));
        props.refresh();
        props.handleClose();
      } else {
        showError(t(message));
      }
    } else {
      const count = parseInt(values.tokenCount, 10) || 1;
      let successCount = 0;
      for (let i = 0; i < count; i++) {
        let { tokenCount: _tc, ...localInputs } = values;
        const baseName =
          values.name.trim() === '' ? 'default' : values.name.trim();
        if (i !== 0 || values.name.trim() === '') {
          localInputs.name = `${baseName}-${generateRandomSuffix()}`;
        } else {
          localInputs.name = baseName;
        }
        const remainResult = normalizeRemainQuotaFields(localInputs);
        if (!remainResult.ok) {
          showError(remainResult.message);
          setLoading(false);
          break;
        }
        localInputs.model_limits = sanitizeModelLimits(
          localInputs.model_limits,
        );
        localInputs.channel_limits = isAdminUser
          ? sanitizeChannelLimits(localInputs.channel_limits)
          : [];

        if (localInputs.expired_time !== -1) {
          let time = Date.parse(localInputs.expired_time);
          if (isNaN(time)) {
            showError(t('过期时间格式错误！'));
            setLoading(false);
            break;
          }
          localInputs.expired_time = Math.ceil(time / 1000);
        }
        localInputs.model_limits = localInputs.model_limits.join(',');
        localInputs.model_limits_enabled = localInputs.model_limits.length > 0;
        localInputs.channel_limits = localInputs.channel_limits.join(',');
        localInputs.channel_limits_enabled =
          localInputs.channel_limits.length > 0;
        const packageResult = normalizePackageFields(localInputs);
        if (!packageResult.ok) {
          showError(packageResult.message);
          setLoading(false);
          break;
        }
        const runtimeLimitResult = normalizeRuntimeLimitFields(localInputs);
        if (!runtimeLimitResult.ok) {
          showError(runtimeLimitResult.message);
          setLoading(false);
          break;
        }
        delete localInputs.remain_amount;
        delete localInputs.package_limit_amount;
        let res = await API.post(`/api/token/`, localInputs);
        const { success, message } = res.data;
        if (success) {
          successCount++;
        } else {
          showError(t(message));
          break;
        }
      }
      if (successCount > 0) {
        showSuccess(t('令牌创建成功，请在列表页面点击复制获取令牌！'));
        props.refresh();
        props.handleClose();
      }
    }
    setLoading(false);
    formApiRef.current?.setValues(getInitValues());
  };

  return (
    <SideSheet
      placement={isEdit ? 'right' : 'left'}
      title={
        <Space>
          {isEdit ? (
            <Tag color='blue' shape='circle'>
              {t('更新')}
            </Tag>
          ) : (
            <Tag color='green' shape='circle'>
              {t('新建')}
            </Tag>
          )}
          <Title heading={4} className='m-0'>
            {isEdit ? t('更新令牌信息') : t('创建新的令牌')}
          </Title>
        </Space>
      }
      bodyStyle={{ padding: '0' }}
      visible={props.visiable}
      width={isMobile ? '100%' : 600}
      footer={
        <div className='flex justify-end bg-white'>
          <Space>
            <Button
              theme='solid'
              className='!rounded-lg'
              onClick={() => formApiRef.current?.submitForm()}
              icon={<IconSave />}
              loading={loading}
            >
              {t('提交')}
            </Button>
            <Button
              theme='light'
              className='!rounded-lg'
              type='primary'
              onClick={handleCancel}
              icon={<IconClose />}
            >
              {t('取消')}
            </Button>
          </Space>
        </div>
      }
      closeIcon={null}
      onCancel={() => handleCancel()}
    >
      <Spin spinning={loading}>
        <Form
          key={isEdit ? 'edit' : 'new'}
          initValues={getInitValues()}
          getFormApi={(api) => {
            formApiRef.current = api;
            applyLoadedTokenValues();
          }}
          onValueChange={(values) => {
            if (Object.prototype.hasOwnProperty.call(values, 'group')) {
              loadModels(values.group || '', true);
              loadChannels(
                values.group || '',
                formApiRef.current?.getValue('model_limits') || [],
                true,
              );
            }
            if (Object.prototype.hasOwnProperty.call(values, 'model_limits')) {
              loadChannels(
                formApiRef.current?.getValue('group') || '',
                values.model_limits || [],
                true,
              );
            }
          }}
          onSubmit={submit}
          onSubmitFail={(errs) => {
            const first = Object.values(errs || {})[0];
            if (first) {
              showError(Array.isArray(first) ? first[0] : first);
            }
            formApiRef.current?.scrollToError();
          }}
        >
          {({ values }) => (
            <div className='p-2'>
              {isSellableToken ? (
                <Card className='!rounded-2xl shadow-sm border-0 mb-4 bg-[var(--semi-color-primary-light-default)]'>
                  <Text className='text-sm'>
                    {t(
                      '当前为可售令牌，仅会保存名称和分组修改，其余额度与限制配置保持只读。',
                    )}
                  </Text>
                </Card>
              ) : null}

              {!isSellableToken && (
                <Card className='!rounded-2xl shadow-sm border-0'>
                  <div className='flex items-center mb-2'>
                    <Avatar
                      size='small'
                      color='orange'
                      className='mr-2 shadow-md'
                    >
                      <IconCreditCard size={16} />
                    </Avatar>
                    <div>
                      <Text className='text-lg font-medium'>
                        {t('创建模式')}
                      </Text>
                      <div className='text-xs text-gray-600'>
                        {t('标准令牌保持现有体验；套餐令牌可按周期配置额度')}
                      </div>
                    </div>
                  </div>
                  <Space wrap>
                    <Button
                      type={tokenMode === 'standard' ? 'primary' : 'tertiary'}
                      theme={tokenMode === 'standard' ? 'solid' : 'light'}
                      onClick={() => switchTokenMode('standard')}
                    >
                      {t('标准令牌')}
                    </Button>
                    <Button
                      type={tokenMode === 'package' ? 'primary' : 'tertiary'}
                      theme={tokenMode === 'package' ? 'solid' : 'light'}
                      onClick={() => switchTokenMode('package')}
                    >
                      {t('套餐令牌')}
                    </Button>
                  </Space>
                </Card>
              )}

              {/* 基础信息 */}
              <Card className='!rounded-2xl shadow-sm border-0'>
                <div className='flex items-center mb-2'>
                  <Avatar size='small' color='blue' className='mr-2 shadow-md'>
                    <IconKey size={16} />
                  </Avatar>
                  <div>
                    <Text className='text-lg font-medium'>{t('基本信息')}</Text>
                    <div className='text-xs text-gray-600'>
                      {t('设置令牌的基本信息')}
                    </div>
                  </div>
                </div>
                <Row gutter={12}>
                  <Col span={24}>
                    <Form.Input
                      field='name'
                      label={t('名称')}
                      placeholder={t('请输入名称')}
                      rules={[{ required: true, message: t('请输入名称') }]}
                      showClear
                    />
                  </Col>
                  <Col span={24}>
                    {groups.length > 0 ? (
                      <Form.Select
                        field='group'
                        label={t('令牌分组')}
                        placeholder={t('请选择令牌分组')}
                        optionList={groups}
                        renderOptionItem={renderGroupOption}
                        rules={[
                          {
                            required: true,
                            message: t('请选择令牌分组'),
                          },
                        ]}
                        style={{ width: '100%' }}
                      />
                    ) : (
                      <Form.Select
                        placeholder={t('管理员未设置用户可选分组')}
                        disabled
                        label={t('令牌分组')}
                        style={{ width: '100%' }}
                      />
                    )}
                  </Col>
                  {!isSellableToken && (
                    <>
                      <Col
                        span={24}
                        style={{
                          display: values.group === 'auto' ? 'block' : 'none',
                        }}
                      >
                        <Form.Switch
                          field='cross_group_retry'
                          label={t('跨分组重试')}
                          size='default'
                          extraText={t(
                            '开启后，当前分组渠道失败时会按顺序尝试下一个分组的渠道',
                          )}
                        />
                      </Col>
                      <Col xs={24} sm={24} md={24} lg={10} xl={10}>
                        <Form.DatePicker
                          field='expired_time'
                          label={t('过期时间')}
                          type='dateTime'
                          placeholder={t('请选择过期时间')}
                          rules={[
                            { required: true, message: t('请选择过期时间') },
                            {
                              validator: (rule, value) => {
                                // 允许 -1 表示永不过期，空值交给必填校验处理。
                                if (value === -1 || !value)
                                  return Promise.resolve();
                                const time = Date.parse(value);
                                if (isNaN(time)) {
                                  return Promise.reject(
                                    t('过期时间格式错误！'),
                                  );
                                }
                                if (time <= Date.now()) {
                                  return Promise.reject(
                                    t('过期时间不能早于当前时间！'),
                                  );
                                }
                                return Promise.resolve();
                              },
                            },
                          ]}
                          showClear
                          style={{ width: '100%' }}
                        />
                      </Col>
                      <Col xs={24} sm={24} md={24} lg={14} xl={14}>
                        <Form.Slot label={t('过期时间快捷设置')}>
                          <Space wrap>
                            <Button
                              theme='light'
                              type='primary'
                              onClick={() => setExpiredTime(0, 0, 0, 0)}
                            >
                              {t('永不过期')}
                            </Button>
                            <Button
                              theme='light'
                              type='tertiary'
                              onClick={() => setExpiredTime(1, 0, 0, 0)}
                            >
                              {t('一个月')}
                            </Button>
                            <Button
                              theme='light'
                              type='tertiary'
                              onClick={() => setExpiredTime(0, 1, 0, 0)}
                            >
                              {t('一天')}
                            </Button>
                            <Button
                              theme='light'
                              type='tertiary'
                              onClick={() => setExpiredTime(0, 0, 1, 0)}
                            >
                              {t('一小时')}
                            </Button>
                          </Space>
                        </Form.Slot>
                      </Col>
                      {!isEdit && (
                        <Col span={24}>
                          <Form.InputNumber
                            field='tokenCount'
                            label={t('新建数量')}
                            min={1}
                            extraText={t(
                              '批量创建时会在名称后自动添加随机后缀',
                            )}
                            rules={[
                              { required: true, message: t('请输入新建数量') },
                            ]}
                            style={{ width: '100%' }}
                          />
                        </Col>
                      )}
                    </>
                  )}
                </Row>
              </Card>

              {!isSellableToken && (
                <>
                  {/* 额度设置 */}
                  <Card className='!rounded-2xl shadow-sm border-0'>
                    <div className='flex items-center mb-2'>
                      <Avatar
                        size='small'
                        color='green'
                        className='mr-2 shadow-md'
                      >
                        <IconCreditCard size={16} />
                      </Avatar>
                      <div>
                        <Text className='text-lg font-medium'>
                          {t('额度设置')}
                        </Text>
                        <div className='text-xs text-gray-600'>
                          {t('设置令牌可用额度和数量')}
                        </div>
                      </div>
                    </div>
                    <Row gutter={12}>
                      <Col span={24}>
                        <Form.InputNumber
                          field='remain_amount'
                          label={t('总额度金额 (USD)')}
                          placeholder={t('例如 500（USD）')}
                          prefix='$'
                          min={0}
                          precision={2}
                          disabled={values.unlimited_quota}
                          rules={
                            values.unlimited_quota
                              ? []
                              : [
                                  {
                                    required: true,
                                    message: t('请输入总额度金额（USD）'),
                                  },
                                ]
                          }
                          extraText={t(
                            '按 USD 输入，仅用于换算，实际保存的是额度',
                          )}
                          style={{ width: '100%' }}
                        />
                      </Col>
                      <Col span={24}>
                        <Form.Switch
                          field='unlimited_quota'
                          label={t('无限额度')}
                          size='default'
                          extraText={t(
                            '令牌的额度仅用于限制令牌本身的最大额度使用量，实际的使用受到账户的剩余额度限制',
                          )}
                        />
                      </Col>
                    </Row>
                  </Card>

                  {tokenMode === 'package' && (
                    <Card className='!rounded-2xl shadow-sm border-0'>
                      <div className='flex items-center mb-2'>
                        <Avatar
                          size='small'
                          color='red'
                          className='mr-2 shadow-md'
                        >
                          <IconCreditCard size={16} />
                        </Avatar>
                        <div>
                          <Text className='text-lg font-medium'>
                            {t('套餐配置')}
                          </Text>
                          <div className='text-xs text-gray-600'>
                            {t('仅设置周期和每周期额度，计费规则保持不变')}
                          </div>
                        </div>
                      </div>
                      <Row gutter={12}>
                        <Col span={24}>
                          <Form.Select
                            field='package_period'
                            label={t('套餐周期')}
                            optionList={[
                              { label: t('每小时'), value: 'hourly' },
                              { label: t('每日'), value: 'daily' },
                              { label: t('每周'), value: 'weekly' },
                              { label: t('每月'), value: 'monthly' },
                              { label: t('自定义'), value: 'custom' },
                            ]}
                            style={{ width: '100%' }}
                          />
                        </Col>
                        <Col
                          span={24}
                          style={{
                            display:
                              values.package_period === 'custom'
                                ? 'none'
                                : 'block',
                          }}
                        >
                          <Form.Select
                            field='package_period_mode'
                            label={t('周期模式')}
                            optionList={[
                              { label: t('相对周期'), value: 'relative' },
                              { label: t('自然周期'), value: 'natural' },
                            ]}
                            extraText={
                              values.package_period_mode === 'natural'
                                ? t('每日/每周一/每月1号 00:00 重置')
                                : t('从激活时间起按固定间隔重置')
                            }
                            style={{ width: '100%' }}
                          />
                        </Col>
                        <Col span={24}>
                          <Form.InputNumber
                            field='package_limit_amount'
                            label={t('周期金额 (USD)')}
                            placeholder={t('例如 10（USD）')}
                            prefix='$'
                            min={0}
                            precision={2}
                            style={{ width: '100%' }}
                            rules={[
                              {
                                required: true,
                                message: t('请输入周期金额（USD）'),
                              },
                            ]}
                          />
                        </Col>
                        <Col
                          span={24}
                          style={{
                            display:
                              values.package_period === 'custom'
                                ? 'block'
                                : 'none',
                          }}
                        >
                          <Form.InputNumber
                            field='package_custom_seconds'
                            label={t('自定义周期秒数')}
                            min={1}
                            extraText={t('例如 86400 表示每天重置')}
                            style={{ width: '100%' }}
                          />
                        </Col>
                      </Row>
                    </Card>
                  )}

                  {/* 访问限制 */}
                  <Card className='!rounded-2xl shadow-sm border-0'>
                    <div className='flex items-center mb-2'>
                      <Avatar
                        size='small'
                        color='purple'
                        className='mr-2 shadow-md'
                      >
                        <IconLink size={16} />
                      </Avatar>
                      <div>
                        <Text className='text-lg font-medium'>
                          {t('访问限制')}
                        </Text>
                        <div className='text-xs text-gray-600'>
                          {t('设置令牌的访问限制')}
                        </div>
                      </div>
                    </div>
                    <Row gutter={12}>
                      <Col span={24}>
                        <Form.Select
                          field='model_limits'
                          label={t('模型限制列表')}
                          placeholder={t(
                            '请选择该令牌支持的模型，留空支持所有模型',
                          )}
                          multiple
                          optionList={models}
                          extraText={t('非必要，不建议启用模型限制')}
                          filter={selectFilter}
                          autoClearSearchValue={false}
                          searchPosition='dropdown'
                          showClear
                          style={{ width: '100%' }}
                        />
                      </Col>
                      {isAdminUser && (
                        <Col span={24}>
                          <Form.Slot
                            label={
                              <div className='flex w-full items-center justify-between gap-3'>
                                <span>{t('渠道限制列表')}</span>
                                <div className='flex items-center gap-2'>
                                  <Text className='text-xs text-gray-500'>
                                    {t('标签聚合显示')}
                                  </Text>
                                  <Switch
                                    size='small'
                                    checked={channelLimitTagMode}
                                    onChange={updateChannelLimitTagMode}
                                  />
                                </div>
                              </div>
                            }
                          >
                            {(() => {
                              const selectedChannelValues =
                                sanitizeChannelLimits(
                                  values.channel_limits || [],
                                );
                              const selectedChannelSet = new Set(
                                selectedChannelValues.map((channelId) =>
                                  String(channelId),
                                ),
                              );
                              const selectedTagItems =
                                channelTagGroups.tagGroups
                                  .map((tagGroup) => {
                                    const selectedCount =
                                      getChannelTagSelectionCount(
                                        selectedChannelSet,
                                        tagGroup.channelIds,
                                      );
                                    if (selectedCount <= 0) {
                                      return null;
                                    }
                                    return {
                                      ...tagGroup,
                                      selectedCount,
                                      selectedChannelIds:
                                        tagGroup.channelIds.filter(
                                          (channelId) =>
                                            selectedChannelSet.has(
                                              String(channelId),
                                            ),
                                        ),
                                    };
                                  })
                                  .filter(Boolean);
                              const selectedUntaggedChannels =
                                channelTagGroups.untaggedChannels.filter(
                                  (option) =>
                                    selectedChannelSet.has(option.value),
                                );

                              if (!channelLimitTagMode) {
                                return (
                                  <Select
                                    value={selectedChannelValues}
                                    placeholder={t(
                                      '请选择该令牌允许使用的渠道，留空表示不限制渠道',
                                    )}
                                    multiple
                                    optionList={channelOptions}
                                    filter={selectFilter}
                                    autoClearSearchValue={false}
                                    searchPosition='dropdown'
                                    renderSelectedItem={(optionNode) => {
                                      const channelName =
                                        optionNode?.channelName ||
                                        optionNode?.value ||
                                        t('未知渠道');
                                      return {
                                        isRenderInTag: true,
                                        content: (
                                          <span
                                            className='cursor-default select-none'
                                            title={
                                              optionNode?.channelSummary ||
                                              channelName
                                            }
                                          >
                                            {channelName}
                                          </span>
                                        ),
                                      };
                                    }}
                                    onChange={(nextValues) =>
                                      applyChannelLimitValues(nextValues || [])
                                    }
                                    showClear
                                    style={{ width: '100%' }}
                                  />
                                );
                              }

                              return (
                                <div className='rounded-xl border border-[var(--semi-color-border)] bg-[var(--semi-color-fill-0)] px-3 py-3'>
                                  {selectedTagItems.length > 0 ||
                                  selectedUntaggedChannels.length > 0 ? (
                                    <div className='mb-3 flex flex-wrap gap-2'>
                                      {selectedTagItems.map((tagGroup) => (
                                        <Tag
                                          key={tagGroup.value}
                                          closable
                                          size='small'
                                          color={
                                            tagGroup.selectedCount ===
                                            tagGroup.channelIds.length
                                              ? 'blue'
                                              : 'orange'
                                          }
                                          shape='circle'
                                          onClose={() =>
                                            removeChannelLimitValues(
                                              tagGroup.selectedChannelIds,
                                            )
                                          }
                                        >
                                          {tagGroup.selectedCount ===
                                          tagGroup.channelIds.length
                                            ? tagGroup.tag
                                            : `${tagGroup.tag}（已选 ${tagGroup.selectedCount}/${tagGroup.channelIds.length}）`}
                                        </Tag>
                                      ))}
                                      {selectedUntaggedChannels.map(
                                        (option) => (
                                          <Tag
                                            key={option.value}
                                            closable
                                            size='small'
                                            color='cyan'
                                            shape='circle'
                                            onClose={() =>
                                              removeChannelLimitValues([
                                                option.value,
                                              ])
                                            }
                                          >
                                            {option.channelName}
                                          </Tag>
                                        ),
                                      )}
                                    </div>
                                  ) : (
                                    <div className='mb-3 text-xs text-gray-500'>
                                      {t('当前未选择任何渠道限制')}
                                    </div>
                                  )}

                                  <div className='flex max-h-64 flex-col gap-2 overflow-y-auto pr-1'>
                                    {channelTagGroups.tagGroups.map(
                                      (tagGroup) => {
                                        const selectedCount =
                                          getChannelTagSelectionCount(
                                            selectedChannelSet,
                                            tagGroup.channelIds,
                                          );
                                        return (
                                          <div
                                            key={tagGroup.value}
                                            className='rounded-xl border border-[var(--semi-color-border)] bg-white px-3 py-2'
                                          >
                                            <Checkbox
                                              checked={
                                                selectedCount ===
                                                  tagGroup.channelIds.length &&
                                                tagGroup.channelIds.length > 0
                                              }
                                              indeterminate={
                                                selectedCount > 0 &&
                                                selectedCount <
                                                  tagGroup.channelIds.length
                                              }
                                              onChange={(e) =>
                                                toggleChannelTagGroup(
                                                  tagGroup,
                                                  e.target.checked,
                                                )
                                              }
                                            >
                                              <div className='flex flex-col gap-1'>
                                                <span className='text-sm font-medium'>
                                                  {tagGroup.tag}
                                                </span>
                                                <span className='text-xs text-gray-500'>
                                                  {selectedCount > 0
                                                    ? `${t('已选')} ${selectedCount}/${tagGroup.channelIds.length} | `
                                                    : ''}
                                                  {getChannelTagSummary(
                                                    tagGroup,
                                                  )}
                                                </span>
                                              </div>
                                            </Checkbox>
                                          </div>
                                        );
                                      },
                                    )}

                                    {channelTagGroups.untaggedChannels.map(
                                      (option) => (
                                        <div
                                          key={option.value}
                                          className='rounded-xl border border-[var(--semi-color-border)] bg-white px-3 py-2'
                                        >
                                          <Checkbox
                                            checked={selectedChannelSet.has(
                                              option.value,
                                            )}
                                            onChange={(e) =>
                                              toggleSingleChannelLimit(
                                                option.value,
                                                e.target.checked,
                                              )
                                            }
                                          >
                                            <div className='flex flex-col gap-1'>
                                              <span className='text-sm font-medium'>
                                                {option.channelName}
                                              </span>
                                              {option.channelSummary ? (
                                                <span className='text-xs text-gray-500'>
                                                  {option.channelSummary}
                                                </span>
                                              ) : null}
                                            </div>
                                          </Checkbox>
                                        </div>
                                      ),
                                    )}

                                    {channelOptions.length === 0 ? (
                                      <div className='py-2 text-sm text-gray-500'>
                                        {t('当前分组和模型限制下暂无可选渠道')}
                                      </div>
                                    ) : null}
                                  </div>

                                  {channelTagGroups.tagGroups.length > 0 ? (
                                    <div className='mt-3 text-xs text-gray-500'>
                                      {t(
                                        '勾选标签会一次性选中该标签下全部渠道；关闭已选标签会移除该标签下当前已选渠道',
                                      )}
                                    </div>
                                  ) : null}
                                </div>
                              );
                            })()}
                            <div className='mt-1 text-xs text-gray-600'>
                              {t(
                                '仅管理员可配置；可选渠道会随分组和模型限制自动联动',
                              )}
                            </div>
                          </Form.Slot>
                        </Col>
                      )}
                      <Col span={24}>
                        <Form.TextArea
                          field='allow_ips'
                          label={t('IP白名单（支持CIDR表达式）')}
                          placeholder={t('允许的IP，一行一个，不填写则不限制')}
                          autosize
                          rows={1}
                          extraText={t(
                            '请勿过度信任此功能，IP可能被伪造，请配合nginx和cdn等网关使用',
                          )}
                          showClear
                          style={{ width: '100%' }}
                        />
                      </Col>
                      <Col span={8}>
                        <Form.InputNumber
                          field='max_concurrency'
                          label={t('并发上限')}
                          min={0}
                          style={{ width: '100%' }}
                          extraText={t('0 表示不限制')}
                        />
                      </Col>
                      <Col span={8}>
                        <Form.InputNumber
                          field='window_request_limit'
                          label={t('窗口请求上限')}
                          min={0}
                          style={{ width: '100%' }}
                          extraText={t('0 表示不限制')}
                        />
                      </Col>
                      <Col span={8}>
                        <Form.InputNumber
                          field='window_seconds'
                          label={t('窗口时长（秒）')}
                          min={0}
                          style={{ width: '100%' }}
                          extraText={t('与窗口请求上限配合使用，0 表示不限制')}
                        />
                      </Col>
                    </Row>
                  </Card>
                </>
              )}
            </div>
          )}
        </Form>
      </Spin>
    </SideSheet>
  );
};

export default EditTokenModal;
