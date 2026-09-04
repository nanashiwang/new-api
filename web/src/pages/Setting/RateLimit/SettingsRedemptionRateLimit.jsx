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

import React, { useEffect, useRef, useState } from 'react';
import {
  Button,
  Col,
  Form,
  InputNumber,
  Modal,
  Row,
  Select,
  Spin,
  Typography,
} from '@douyinfe/semi-ui';
import {
  compareObjects,
  API,
  showError,
  showSuccess,
  showWarning,
} from '../../../helpers';
import { useTranslation } from 'react-i18next';
import {
  REDEMPTION_LIMIT_DEFAULTS,
  resolveRedemptionLimitInputs,
  validateRedemptionPolicyInputs,
} from './redemptionPolicy';

const UNIT_SECONDS = { second: 1, minute: 60, hour: 3600 };

function pickDisplayUnit(seconds) {
  const n = Number(seconds) || 0;
  if (n > 0 && n % 3600 === 0) return 'hour';
  if (n > 0 && n % 60 === 0) return 'minute';
  return 'second';
}

function secondsToDisplay(seconds, unit) {
  const n = Number(seconds) || 0;
  const divisor = UNIT_SECONDS[unit] || 1;
  return Math.max(1, Math.round(n / divisor));
}

export default function RedemptionRateLimit(props) {
  const { t } = useTranslation();

  const [loading, setLoading] = useState(false);
  const [inputs, setInputs] = useState({ ...REDEMPTION_LIMIT_DEFAULTS });
  const [inputsRow, setInputsRow] = useState(inputs);
  const [unit, setUnit] = useState('minute');
  const [durationDisplay, setDurationDisplay] = useState(10);
  const refForm = useRef();

  function onDurationDisplayChange(value) {
    const v = Math.max(1, Number(value) || 1);
    setDurationDisplay(v);
    const secs = v * (UNIT_SECONDS[unit] || 1);
    setInputs((prev) => ({
      ...prev,
      RedemptionRateLimitDurationSeconds: String(secs),
    }));
  }

  function onUnitChange(newUnit) {
    const displayValue = Math.max(1, Number(durationDisplay) || 1);
    setUnit(newUnit);
    setInputs((prev) => ({
      ...prev,
      RedemptionRateLimitDurationSeconds: String(
        displayValue * (UNIT_SECONDS[newUnit] || 1),
      ),
    }));
  }

  function onSubmit() {
    const validation = validateRedemptionPolicyInputs(inputs);
    if (validation.errorKey) return showError(t(validation.errorKey));
    const { minimumQuota, activeLimit, creatorThreshold, smallQuotaLimit } =
      validation.values;
    const updateArray = compareObjects(inputs, inputsRow);
    if (!updateArray.length) return showWarning(t('你似乎并没有修改什么'));
    const policyKeys = new Set([
      'WalletRedemptionDailyCreateLimit',
      'WalletRedemptionMinimumQuota',
      'WalletRedemptionActiveLimit',
      'WalletRedemptionDailyQuotaLimit',
      'WalletRedemptionReviewDistinctCreatorThreshold',
      'WalletRedemptionReviewSmallQuotaLimit',
    ]);
    const requestFactories = updateArray
      .filter((item) => !policyKeys.has(item.key))
      .map(
        (item) => () =>
          API.put('/api/option/', {
            key: item.key,
            value: String(inputs[item.key]),
          }),
      );
    if (updateArray.some((item) => policyKeys.has(item.key))) {
      requestFactories.push(() =>
        API.put('/api/option/', {
          key: 'WalletRedemptionPolicyBundle',
          value: JSON.stringify({
            daily_create_limit: Number(inputs.WalletRedemptionDailyCreateLimit),
            minimum_quota: minimumQuota,
            active_limit: activeLimit,
            daily_quota_limit: Number(inputs.WalletRedemptionDailyQuotaLimit),
            review_distinct_creator_threshold: creatorThreshold,
            review_small_quota_limit: smallQuotaLimit,
          }),
        }),
      );
    }
    const save = () => {
      const requestQueue = requestFactories.map((request) => request());
      setLoading(true);
      return Promise.all(requestQueue)
        .then((res) => {
          if (requestQueue.length === 1) {
            if (res.includes(undefined)) return;
          } else if (requestQueue.length > 1) {
            if (res.includes(undefined))
              return showError(t('部分保存失败，请重试'));
          }
          for (let i = 0; i < res.length; i++) {
            if (!res[i].data.success) {
              return showError(res[i].data.message);
            }
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
    };
    const policyChanged = updateArray.some((item) => policyKeys.has(item.key));
    const disablesProtection =
      Number(inputs.WalletRedemptionDailyCreateLimit) === 0 ||
      minimumQuota === 0 ||
      activeLimit === 0 ||
      Number(inputs.WalletRedemptionDailyQuotaLimit) === 0 ||
      creatorThreshold === 0 ||
      smallQuotaLimit === 0;
    if (policyChanged && disablesProtection) {
      Modal.confirm({
        title: t('确认保存宽松的兑换码策略？'),
        content: t('当前配置包含 0，将关闭对应限制或风控，请确认风险可接受。'),
        onOk: save,
      });
      return;
    }
    save();
  }

  useEffect(() => {
    const currentInputs = resolveRedemptionLimitInputs(props.options);
    setInputs(currentInputs);
    setInputsRow(structuredClone(currentInputs));
    const nextUnit = pickDisplayUnit(
      currentInputs.RedemptionRateLimitDurationSeconds,
    );
    setUnit(nextUnit);
    setDurationDisplay(
      secondsToDisplay(
        currentInputs.RedemptionRateLimitDurationSeconds,
        nextUnit,
      ),
    );
    if (refForm.current) {
      refForm.current.setValues(currentInputs);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [props.options]);

  return (
    <>
      <Spin spinning={loading}>
        <Form
          values={inputs}
          getFormApi={(formAPI) => (refForm.current = formAPI)}
          style={{ marginBottom: 15 }}
        >
          <Form.Section text={t('兑换码速率限制')}>
            <Row gutter={16}>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Switch
                  field={'RedemptionRateLimitEnabled'}
                  label={t('启用兑换码兑换速率限制')}
                  size='default'
                  checkedText='｜'
                  uncheckedText='〇'
                  onChange={(value) => {
                    setInputs({
                      ...inputs,
                      RedemptionRateLimitEnabled: value,
                    });
                  }}
                />
              </Col>
            </Row>
            <Row gutter={16} style={{ marginBottom: 12 }}>
              <Col xs={24} sm={12} md={6} lg={6} xl={6}>
                <Typography.Text strong>{t('限制周期')}</Typography.Text>
                <InputNumber
                  style={{ width: '100%', marginTop: 4 }}
                  min={1}
                  step={1}
                  value={durationDisplay}
                  onChange={onDurationDisplayChange}
                />
                <Typography.Text type='tertiary' size='small'>
                  {t('搭配右侧“单位”使用，后端会换算为秒后保存')}
                </Typography.Text>
              </Col>
              <Col xs={24} sm={12} md={6} lg={6} xl={6}>
                <Typography.Text strong>{t('单位')}</Typography.Text>
                <Select
                  style={{ width: '100%', marginTop: 4 }}
                  value={unit}
                  onChange={onUnitChange}
                  optionList={[
                    { label: t('秒'), value: 'second' },
                    { label: t('分钟'), value: 'minute' },
                    { label: t('小时'), value: 'hour' },
                  ]}
                />
              </Col>
            </Row>
            <Row gutter={16}>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.InputNumber
                  label={t('每周期最多成功兑换次数')}
                  step={1}
                  min={0}
                  max={100000000}
                  suffix={t('次')}
                  extraText={t('0 代表不限制')}
                  field={'RedemptionRateLimitSuccessCount'}
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      RedemptionRateLimitSuccessCount: String(value),
                    })
                  }
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.InputNumber
                  label={t('每周期最多失败次数')}
                  step={1}
                  min={0}
                  max={100000000}
                  suffix={t('次')}
                  extraText={t('0 代表不限制；到达后锁定整个周期')}
                  field={'RedemptionRateLimitFailureCount'}
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      RedemptionRateLimitFailureCount: String(value),
                    })
                  }
                />
              </Col>
            </Row>
          </Form.Section>
          <Form.Section text={t('用户创建兑换码限制')}>
            <Row gutter={16}>
              <Col xs={24} sm={12} md={4} lg={4} xl={4}>
                <Form.InputNumber
                  label={t('单日最多创建数量')}
                  step={1}
                  min={0}
                  suffix={t('个')}
                  extraText={t('0 代表不限制；按北京时间自然日统计')}
                  field={'WalletRedemptionDailyCreateLimit'}
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      WalletRedemptionDailyCreateLimit: String(value),
                    })
                  }
                />
              </Col>
              <Col xs={24} sm={12} md={4} lg={4} xl={4}>
                <Form.InputNumber
                  label={t('单日最多转赠额度')}
                  step={1}
                  min={0}
                  suffix={t('闪电')}
                  extraText={t('0 代表不限制；按北京时间自然日统计')}
                  field={'WalletRedemptionDailyQuotaLimit'}
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      WalletRedemptionDailyQuotaLimit: String(value),
                    })
                  }
                />
              </Col>
              <Col xs={24} sm={12} md={4} lg={4} xl={4}>
                <Form.InputNumber
                  label={t('单个兑换码最低额度')}
                  step={1}
                  min={0}
                  suffix={t('闪电')}
                  extraText={t('0 代表不限制；管理员不受该限制')}
                  field={'WalletRedemptionMinimumQuota'}
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      WalletRedemptionMinimumQuota: String(value),
                    })
                  }
                />
              </Col>
              <Col xs={24} sm={12} md={4} lg={4} xl={4}>
                <Form.InputNumber
                  label={t('最多保留未使用兑换码')}
                  step={1}
                  min={0}
                  suffix={t('个')}
                  extraText={t('0 代表不限制')}
                  field={'WalletRedemptionActiveLimit'}
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      WalletRedemptionActiveLimit: String(value),
                    })
                  }
                />
              </Col>
              <Col xs={24} sm={12} md={4} lg={4} xl={4}>
                <Form.InputNumber
                  label={t('触发复查的不同创建人数')}
                  step={1}
                  min={0}
                  suffix={t('人')}
                  extraText={t(
                    '同一用户单日达到该来源数后进入人工复查；0 关闭',
                  )}
                  field={'WalletRedemptionReviewDistinctCreatorThreshold'}
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      WalletRedemptionReviewDistinctCreatorThreshold:
                        String(value),
                    })
                  }
                />
              </Col>
              <Col xs={24} sm={12} md={4} lg={4} xl={4}>
                <Form.InputNumber
                  label={t('纳入复查的小额码上限')}
                  step={1}
                  min={0}
                  suffix={t('闪电')}
                  extraText={t('只统计不高于该额度的钱包兑换码；0 关闭')}
                  field={'WalletRedemptionReviewSmallQuotaLimit'}
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      WalletRedemptionReviewSmallQuotaLimit: String(value),
                    })
                  }
                />
              </Col>
              <Col xs={24} sm={12} md={4} lg={4} xl={4}>
                <Form.InputNumber
                  label={t('钱包兑换码自动绑定账号最大年龄')}
                  step={1}
                  min={0}
                  suffix={t('小时')}
                  extraText={t('仅影响钱包兑换码自动建立邀请关系；0 表示关闭')}
                  field={'WalletRedemptionAutoBindMaxAgeHours'}
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      WalletRedemptionAutoBindMaxAgeHours: String(value),
                    })
                  }
                />
              </Col>
            </Row>
            <Row>
              <Button size='default' onClick={onSubmit}>
                {t('保存兑换码限制')}
              </Button>
            </Row>
          </Form.Section>
        </Form>
      </Spin>
    </>
  );
}
