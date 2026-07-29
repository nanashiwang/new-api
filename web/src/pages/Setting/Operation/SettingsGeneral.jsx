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

import React, { useEffect, useState, useRef, useMemo } from 'react';
import {
  Banner,
  Button,
  Col,
  Form,
  Row,
  Spin,
  Modal,
  Select,
  InputGroup,
  Input,
} from '@douyinfe/semi-ui';
import {
  compareObjects,
  API,
  showError,
  showSuccess,
  showWarning,
} from '../../../helpers';
import { useTranslation } from 'react-i18next';

const DEFAULT_INVITE_BINDING_SETTINGS = {
  threshold: 0,
  rate_after_threshold: 100,
};

function parseInviteBindingSettings(raw) {
  try {
    const parsed = typeof raw === 'string' ? JSON.parse(raw) : raw;
    const threshold = Number(parsed?.threshold);
    const rate = Number(parsed?.rate_after_threshold);
    if (
      !Number.isSafeInteger(threshold) ||
      threshold < 0 ||
      !Number.isInteger(rate) ||
      rate < 0 ||
      rate > 100
    ) {
      return DEFAULT_INVITE_BINDING_SETTINGS;
    }
    return { threshold, rate_after_threshold: rate };
  } catch {
    return DEFAULT_INVITE_BINDING_SETTINGS;
  }
}

function hasNumericInput(value) {
  return value !== '' && value !== null && value !== undefined;
}

export default function GeneralSettings(props) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [showQuotaWarning, setShowQuotaWarning] = useState(false);
  const [inputs, setInputs] = useState({
    TopUpLink: '',
    'general_setting.docs_link': '',
    'general_setting.quota_display_type': 'USD',
    'general_setting.custom_currency_symbol': '¤',
    'general_setting.custom_currency_exchange_rate': '',
    'general_setting.payment_currency_symbol': '¥',
    QuotaPerUnit: '',
    RetryTimes: '',
    ResponsesRequestBodyLimitMB: '',
    USDExchangeRate: '',
    DisplayTokenStatEnabled: false,
    DefaultCollapseSidebar: false,
    DemoSiteEnabled: false,
    SelfUseModeEnabled: false,
    'token_setting.max_user_tokens': 1000,
    InviteBindingThreshold: 0,
    InviteBindingRateAfterThreshold: 100,
  });
  const refForm = useRef();
  const [inputsRow, setInputsRow] = useState(inputs);

  function handleFieldChange(fieldName) {
    return (value) => {
      setInputs((inputs) => ({ ...inputs, [fieldName]: value }));
    };
  }

  function onSubmit() {
    const updateArray = compareObjects(inputs, inputsRow);
    if (!updateArray.length) return showWarning(t('你似乎并没有修改什么'));
    const inviteBindingFields = new Set([
      'InviteBindingThreshold',
      'InviteBindingRateAfterThreshold',
    ]);
    const inviteBindingChanged = updateArray.some((item) =>
      inviteBindingFields.has(item.key),
    );
    const threshold = Number(inputs.InviteBindingThreshold);
    const rate = Number(inputs.InviteBindingRateAfterThreshold);
    if (
      inviteBindingChanged &&
      (!hasNumericInput(inputs.InviteBindingThreshold) ||
        !hasNumericInput(inputs.InviteBindingRateAfterThreshold) ||
        !Number.isSafeInteger(threshold) ||
        threshold < 0 ||
        !Number.isInteger(rate) ||
        rate < 0 ||
        rate > 100)
    ) {
      return showError(
        t(
          '请输入有效的邀请绑定设置：人数阈值为非负整数，绑定成功率为 0 到 100 的整数',
        ),
      );
    }
    const requestQueue = updateArray
      .filter((item) => !inviteBindingFields.has(item.key))
      .map((item) => {
        let value = '';
        if (typeof inputs[item.key] === 'boolean') {
          value = String(inputs[item.key]);
        } else {
          value = inputs[item.key];
        }
        return API.put('/api/option/', {
          key: item.key,
          value,
        });
      });
    if (inviteBindingChanged) {
      requestQueue.push(
        API.put('/api/option/', {
          key: 'InviteBindingSettings',
          value: JSON.stringify({
            threshold,
            rate_after_threshold: rate,
          }),
        }),
      );
    }
    setLoading(true);
    Promise.all(requestQueue)
      .then((res) => {
        if (requestQueue.length === 1) {
          if (res.includes(undefined)) return;
        } else if (requestQueue.length > 1) {
          if (res.includes(undefined))
            return showError(t('部分保存失败，请重试'));
        }
        showSuccess(t('保存成功'));
        props.refresh();
      })
      .catch(() => {
        showError(t('保存失败，请重试'));
      })
      .finally(() => {
        setLoading(false);
      });
  }

  // 计算展示在输入框中的“1 USD = X <currency>”中的 X
  const combinedRate = useMemo(() => {
    const type = inputs['general_setting.quota_display_type'];
    if (type === 'USD') return '1';
    if (type === 'CNY') return String(inputs['USDExchangeRate'] || '');
    if (type === 'TOKENS') return String(inputs['QuotaPerUnit'] || '');
    if (type === 'CUSTOM')
      return String(
        inputs['general_setting.custom_currency_exchange_rate'] || '',
      );
    return '';
  }, [inputs]);

  const onCombinedRateChange = (val) => {
    const type = inputs['general_setting.quota_display_type'];
    if (type === 'CNY') {
      handleFieldChange('USDExchangeRate')(val);
    } else if (type === 'TOKENS') {
      handleFieldChange('QuotaPerUnit')(val);
    } else if (type === 'CUSTOM') {
      handleFieldChange('general_setting.custom_currency_exchange_rate')(val);
    }
  };

  const inviteBindingPreview = useMemo(() => {
    const threshold = Number(inputs.InviteBindingThreshold);
    const rate = Number(inputs.InviteBindingRateAfterThreshold);
    if (
      !hasNumericInput(inputs.InviteBindingThreshold) ||
      !hasNumericInput(inputs.InviteBindingRateAfterThreshold) ||
      !Number.isSafeInteger(threshold) ||
      threshold < 0 ||
      !Number.isInteger(rate) ||
      rate < 0 ||
      rate > 100
    ) {
      return t('当前设置无效，请输入非负整数人数和 0 到 100 的整数成功率。');
    }
    if (threshold === 0) {
      return t('当前规则：邀请关系概率限制已关闭，所有有效邀请都会正常绑定。');
    }
    return t(
      '当前规则：每位用户前 {{threshold}} 名受邀注册用户必定绑定；从第 {{next}} 名开始，每次注册有 {{rate}}% 概率绑定。未绑定用户仍可正常注册，但双方不会获得邀请奖励，也不会产生后续充值返佣。',
      { threshold, next: threshold + 1, rate },
    );
  }, [
    inputs.InviteBindingThreshold,
    inputs.InviteBindingRateAfterThreshold,
    t,
  ]);

  useEffect(() => {
    const currentInputs = {};
    for (let key in props.options) {
      if (Object.keys(inputs).includes(key)) {
        currentInputs[key] = props.options[key];
      }
    }
    // 若旧字段存在且新字段缺失，则做一次兜底映射
    if (
      currentInputs['general_setting.quota_display_type'] === undefined &&
      props.options?.DisplayInCurrencyEnabled !== undefined
    ) {
      currentInputs['general_setting.quota_display_type'] = props.options
        .DisplayInCurrencyEnabled
        ? 'USD'
        : 'TOKENS';
    }
    // 回填自定义货币相关字段（如果后端已存在）
    if (props.options['general_setting.custom_currency_symbol'] !== undefined) {
      currentInputs['general_setting.custom_currency_symbol'] =
        props.options['general_setting.custom_currency_symbol'];
    }
    if (
      props.options['general_setting.custom_currency_exchange_rate'] !==
      undefined
    ) {
      currentInputs['general_setting.custom_currency_exchange_rate'] =
        props.options['general_setting.custom_currency_exchange_rate'];
    }
    if (
      props.options['general_setting.payment_currency_symbol'] !== undefined
    ) {
      currentInputs['general_setting.payment_currency_symbol'] =
        props.options['general_setting.payment_currency_symbol'];
    }
    const inviteBindingSettings = parseInviteBindingSettings(
      props.options?.InviteBindingSettings,
    );
    currentInputs.InviteBindingThreshold = inviteBindingSettings.threshold;
    currentInputs.InviteBindingRateAfterThreshold =
      inviteBindingSettings.rate_after_threshold;
    setInputs(currentInputs);
    setInputsRow(structuredClone(currentInputs));
    refForm.current.setValues(currentInputs);
  }, [props.options]);

  return (
    <>
      <Spin spinning={loading}>
        <Form
          values={inputs}
          getFormApi={(formAPI) => (refForm.current = formAPI)}
          style={{ marginBottom: 15 }}
        >
          <Form.Section text={t('通用设置')}>
            <Row gutter={16}>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Input
                  field={'TopUpLink'}
                  label={t('充值链接')}
                  initValue={''}
                  placeholder={t('例如发卡网站的购买链接')}
                  onChange={handleFieldChange('TopUpLink')}
                  showClear
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Input
                  field={'general_setting.docs_link'}
                  label={t('文档地址')}
                  initValue={''}
                  placeholder={t('例如 /docs 或 https://docs.newapi.pro')}
                  onChange={handleFieldChange('general_setting.docs_link')}
                  showClear
                />
              </Col>
              {/* 单位美元额度已合入汇率组合控件（TOKENS 模式下编辑），不再单独展示 */}
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Input
                  field={'RetryTimes'}
                  label={t('失败重试次数')}
                  initValue={''}
                  placeholder={t('失败重试次数')}
                  onChange={handleFieldChange('RetryTimes')}
                  showClear
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.InputNumber
                  field={'ResponsesRequestBodyLimitMB'}
                  label={t('对话请求体预检上限')}
                  min={0}
                  step={1}
                  suffix='MiB'
                  placeholder='20'
                  extraText={t(
                    '作用于 /v1/responses、/v1/chat/completions 及兼容路径，0 表示关闭业务预检；仍受全局请求体上限、Nginx 和上游限制影响。',
                  )}
                  onChange={handleFieldChange('ResponsesRequestBodyLimitMB')}
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Slot label={t('站点额度展示类型及汇率')}>
                  <InputGroup style={{ width: '100%' }}>
                    <Input
                      prefix={'1 USD = '}
                      style={{ width: '50%' }}
                      value={combinedRate}
                      onChange={onCombinedRateChange}
                      disabled={
                        inputs['general_setting.quota_display_type'] === 'USD'
                      }
                    />
                    <Select
                      style={{ width: '50%' }}
                      value={inputs['general_setting.quota_display_type']}
                      onChange={handleFieldChange(
                        'general_setting.quota_display_type',
                      )}
                    >
                      <Select.Option value='USD'>USD ($)</Select.Option>
                      <Select.Option value='CNY'>CNY (¥)</Select.Option>
                      <Select.Option value='TOKENS'>Tokens</Select.Option>
                      <Select.Option value='CUSTOM'>
                        {t('自定义货币')}
                      </Select.Option>
                    </Select>
                  </InputGroup>
                </Form.Slot>
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Input
                  field={'general_setting.custom_currency_symbol'}
                  label={t('自定义货币符号')}
                  placeholder={t('例如 €, £, Rp, ₩, ₹...')}
                  onChange={handleFieldChange(
                    'general_setting.custom_currency_symbol',
                  )}
                  showClear
                  disabled={
                    inputs['general_setting.quota_display_type'] !== 'CUSTOM'
                  }
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Input
                  field={'general_setting.payment_currency_symbol'}
                  label={t('支付货币符号')}
                  placeholder={t('如 ¥、$、€')}
                  onChange={handleFieldChange(
                    'general_setting.payment_currency_symbol',
                  )}
                  showClear
                />
              </Col>
            </Row>
            <Row gutter={16}>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Switch
                  field={'DisplayTokenStatEnabled'}
                  label={t('额度查询接口返回令牌额度而非用户额度')}
                  size='default'
                  checkedText='｜'
                  uncheckedText='〇'
                  onChange={handleFieldChange('DisplayTokenStatEnabled')}
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Switch
                  field={'DefaultCollapseSidebar'}
                  label={t('默认折叠侧边栏')}
                  size='default'
                  checkedText='｜'
                  uncheckedText='〇'
                  onChange={handleFieldChange('DefaultCollapseSidebar')}
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Switch
                  field={'DemoSiteEnabled'}
                  label={t('演示站点模式')}
                  size='default'
                  checkedText='｜'
                  uncheckedText='〇'
                  onChange={handleFieldChange('DemoSiteEnabled')}
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Switch
                  field={'SelfUseModeEnabled'}
                  label={t('自用模式')}
                  extraText={t('开启后不限制：必须设置模型倍率')}
                  size='default'
                  checkedText='｜'
                  uncheckedText='〇'
                  onChange={handleFieldChange('SelfUseModeEnabled')}
                />
              </Col>
            </Row>
            <Row gutter={16}>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.InputNumber
                  label={t('用户最大令牌数量')}
                  field={'token_setting.max_user_tokens'}
                  step={1}
                  min={1}
                  extraText={t(
                    '每个用户最多可创建的令牌数量，默认 1000，设置过大可能会影响性能',
                  )}
                  placeholder={'1000'}
                  onChange={handleFieldChange('token_setting.max_user_tokens')}
                />
              </Col>
            </Row>
          </Form.Section>
          <Form.Section text={t('邀请关系控制')}>
            <Banner
              type='info'
              description={inviteBindingPreview}
              bordered
              fullMode={false}
              closeIcon={null}
              style={{ marginBottom: 16 }}
            />
            <Row gutter={16}>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.InputNumber
                  field={'InviteBindingThreshold'}
                  label={t('全量绑定人数阈值')}
                  min={0}
                  step={1}
                  precision={0}
                  placeholder='1000'
                  extraText={t('0 表示关闭限制，所有有效邀请都会正常绑定')}
                  onChange={handleFieldChange('InviteBindingThreshold')}
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.InputNumber
                  field={'InviteBindingRateAfterThreshold'}
                  label={t('达到阈值后的绑定成功率')}
                  min={0}
                  max={100}
                  step={1}
                  precision={0}
                  suffix={t('百分比符号')}
                  placeholder='20'
                  extraText={t('仅影响超过阈值后的新注册，不改变已有邀请关系')}
                  onChange={handleFieldChange(
                    'InviteBindingRateAfterThreshold',
                  )}
                />
              </Col>
            </Row>
            <Row>
              <Button size='default' onClick={onSubmit}>
                {t('保存通用设置')}
              </Button>
            </Row>
          </Form.Section>
        </Form>
      </Spin>

      <Modal
        title={t('警告')}
        visible={showQuotaWarning}
        onOk={() => setShowQuotaWarning(false)}
        onCancel={() => setShowQuotaWarning(false)}
        closeOnEsc={true}
        width={500}
      >
        <Banner
          type='warning'
          description={t(
            '此设置用于系统内部计算，默认值500000是为了精确到6位小数点设计，不推荐修改。',
          )}
          bordered
          fullMode={false}
          closeIcon={null}
        />
      </Modal>
    </>
  );
}
