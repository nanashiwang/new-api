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
  const [inputs, setInputs] = useState({
    RedemptionRateLimitEnabled: false,
    RedemptionRateLimitDurationSeconds: 600,
    RedemptionRateLimitSuccessCount: 0,
    RedemptionRateLimitFailureCount: 0,
  });
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
    const updateArray = compareObjects(inputs, inputsRow);
    if (!updateArray.length) return showWarning(t('你似乎并没有修改什么'));
    const requestQueue = updateArray.map((item) =>
      API.put('/api/option/', {
        key: item.key,
        value: String(inputs[item.key]),
      }),
    );
    setLoading(true);
    Promise.all(requestQueue)
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
  }

  useEffect(() => {
    const currentInputs = {};
    for (let key in props.options) {
      if (Object.keys(inputs).includes(key)) {
        currentInputs[key] = props.options[key];
      }
    }
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
                  style={{ minWidth: 56 }}
                  checkedText={t('启用')}
                  uncheckedText={t('关闭')}
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
            <Row>
              <Button size='default' onClick={onSubmit}>
                {t('保存兑换码速率限制')}
              </Button>
            </Row>
          </Form.Section>
        </Form>
      </Spin>
    </>
  );
}
