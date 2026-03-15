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

import React, { useEffect, useState, useContext, useRef } from 'react';
import {
  API,
  showError,
  showInfo,
  showSuccess,
  timestamp2string,
  renderGroupOption,
  getModelCategories,
  selectFilter,
} from '../../../../helpers';
import {
  quotaToUSDAmount,
  usdAmountToQuota,
} from '../../../../helpers/quota';
import { useIsMobile } from '../../../../hooks/common/useIsMobile';
import {
  Button,
  SideSheet,
  Space,
  Spin,
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

const EditTokenModal = (props) => {
  const { t } = useTranslation();
  const [statusState, statusDispatch] = useContext(StatusContext);
  const [loading, setLoading] = useState(false);
  const isMobile = useIsMobile();
  const formApiRef = useRef(null);
  const loadedTokenValuesRef = useRef(null);
  const [models, setModels] = useState([]);
  const [groups, setGroups] = useState([]);
  const [tokenMode, setTokenMode] = useState('standard');
  const isEdit = props.editingToken.id !== undefined;

  const getInitValues = () => ({
    name: '',
    remain_quota: 0,
    remain_amount: 0,
    expired_time: -1,
    unlimited_quota: true,
    model_limits_enabled: false,
    model_limits: [],
    allow_ips: '',
    group: '',
    cross_group_retry: false,
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
      formApiRef.current.setValue('package_limit_amount', values.package_limit_amount);
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
    const nextValues = currentValues.filter((model) => allowedValues.has(model));
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

  const loadModels = async (groupValue = '', shouldNotify = false) => {
    let res = await API.get(`/api/token/models`, {
      params: {
        group: groupValue,
      },
    });
    const { success, message, data } = res.data;
    if (success) {
      const localModelOptions = buildModelOptions(Array.isArray(data) ? data : []);
      setModels(localModelOptions);
      syncModelLimitsWithOptions(localModelOptions, shouldNotify);
    } else {
      setModels([]);
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
      setGroups(localGroupOptions);
      // if (statusState?.status?.default_use_auto_group && formApiRef.current) {
      //   formApiRef.current.setValue('group', 'auto');
      // }
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
      if (!data.package_period || data.package_period === 'none') {
        data.package_period = 'daily';
      }
      data.remain_amount = quotaToUSDAmount(data.remain_quota || 0);
      data.package_limit_amount = quotaToUSDAmount(data.package_limit_quota || 0);
      loadedTokenValuesRef.current = { ...getInitValues(), ...data };
      applyLoadedTokenValues();
      await loadModels(data.group || '');
      setTokenMode(data.package_enabled ? 'package' : 'standard');
    } else {
      showError(message);
    }
    setLoading(false);
  };

  useEffect(() => {
    loadedTokenValuesRef.current = null;
    if (formApiRef.current) {
      if (!isEdit) {
        formApiRef.current.setValues(getInitValues());
      }
    }
    loadModels(props.editingToken.group || '');
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
      }
    } else {
      loadedTokenValuesRef.current = null;
      formApiRef.current?.reset();
    }
  }, [props.visiable, props.editingToken.id]);

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
      const currentLimit = Number(formApiRef.current.getValue('package_limit_quota') || 0);
      if (currentLimit <= 0) {
        formApiRef.current.setValue('package_limit_amount', 10);
        formApiRef.current.setValue('package_limit_quota', usdAmountToQuota(10));
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
    localInputs.package_limit_amount = Number(localInputs.package_limit_amount || 0);
    localInputs.package_limit_quota = usdAmountToQuota(localInputs.package_limit_amount);
    localInputs.package_used_quota = parseInt(localInputs.package_used_quota, 10) || 0;
    localInputs.package_next_reset_time = parseInt(localInputs.package_next_reset_time, 10) || 0;
    const period = (localInputs.package_period || '').trim();
    if (!['daily', 'weekly', 'monthly', 'custom'].includes(period)) {
      return { ok: false, message: t('套餐周期无效') };
    }
    localInputs.package_period = period;
    // custom 周期本身就是相对的，强制设为 relative
    if (period === 'custom') {
      localInputs.package_period_mode = 'relative';
    } else {
      const mode = (localInputs.package_period_mode || '').trim();
      localInputs.package_period_mode = ['relative', 'natural'].includes(mode) ? mode : 'relative';
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

  const submit = async (values) => {
    setLoading(true);
    if (isEdit) {
      let { tokenCount: _tc, ...localInputs } = values;
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
      localInputs.model_limits = sanitizeModelLimits(localInputs.model_limits);
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
        localInputs.model_limits = sanitizeModelLimits(localInputs.model_limits);

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
        const packageResult = normalizePackageFields(localInputs);
        if (!packageResult.ok) {
          showError(packageResult.message);
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
              <Card className='!rounded-2xl shadow-sm border-0'>
                <div className='flex items-center mb-2'>
                  <Avatar size='small' color='orange' className='mr-2 shadow-md'>
                    <IconCreditCard size={16} />
                  </Avatar>
                  <div>
                    <Text className='text-lg font-medium'>{t('创建模式')}</Text>
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
                        placeholder={t('令牌分组，默认为用户的分组')}
                        optionList={groups}
                        renderOptionItem={renderGroupOption}
                        showClear
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
                              return Promise.reject(t('过期时间格式错误！'));
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
                        extraText={t('批量创建时会在名称后自动添加随机后缀')}
                        rules={[
                          { required: true, message: t('请输入新建数量') },
                        ]}
                        style={{ width: '100%' }}
                      />
                    </Col>
                  )}
                </Row>
              </Card>

              {/* 额度设置 */}
              <Card className='!rounded-2xl shadow-sm border-0'>
                <div className='flex items-center mb-2'>
                  <Avatar size='small' color='green' className='mr-2 shadow-md'>
                    <IconCreditCard size={16} />
                  </Avatar>
                  <div>
                    <Text className='text-lg font-medium'>{t('额度设置')}</Text>
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
                          : [{ required: true, message: t('请输入总额度金额（USD）') }]
                      }
                      extraText={t('按 USD 输入，仅用于换算，实际保存的是额度')}
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
                    <Avatar size='small' color='red' className='mr-2 shadow-md'>
                      <IconCreditCard size={16} />
                    </Avatar>
                    <div>
                      <Text className='text-lg font-medium'>{t('套餐配置')}</Text>
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
                        display: values.package_period === 'custom' ? 'none' : 'block',
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
                        rules={[{ required: true, message: t('请输入周期金额（USD）') }]}
                      />
                    </Col>
                    <Col
                      span={24}
                      style={{
                        display: values.package_period === 'custom' ? 'block' : 'none',
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
                    <Text className='text-lg font-medium'>{t('访问限制')}</Text>
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
                </Row>
              </Card>
            </div>
          )}
        </Form>
      </Spin>
    </SideSheet>
  );
};

export default EditTokenModal;
