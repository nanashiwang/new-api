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
import { Button, Card, Form, Typography } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';

import { API, showError, showSuccess, verifyJSON } from '../../../helpers';

const { Paragraph, Text } = Typography;

const DEFAULT_EXAMPLE = `[
  {
    "id": "offpeak-night",
    "enabled": true,
    "timezone": "Asia/Shanghai",
    "start": "00:00",
    "end": "08:00",
    "days": ["mon", "tue", "wed", "thu", "fri", "sat", "sun"],
    "ratio": 0.8,
    "models": ["*"],
    "groups": ["*"],
    "priority": 10
  },
  {
    "id": "peak-evening",
    "enabled": true,
    "timezone": "Asia/Shanghai",
    "start": "18:00",
    "end": "23:30",
    "ratio": 1.2,
    "models": ["gpt-*", "claude-*"],
    "groups": ["default"],
    "priority": 20
  }
]`;

export default function TimeRatioSettings({ options, refresh }) {
  const { t } = useTranslation();
  const formRef = useRef();
  const [loading, setLoading] = useState(false);
  const [rules, setRules] = useState('[]');

  useEffect(() => {
    const nextRules = options?.TimeRatioRules || '[]';
    setRules(nextRules);
    formRef.current?.setValues({ TimeRatioRules: nextRules });
  }, [options]);

  const validateRules = (value) => {
    const raw = value && value.trim() ? value : '[]';
    if (!verifyJSON(raw)) {
      return false;
    }
    try {
      const parsed = JSON.parse(raw);
      return Array.isArray(parsed);
    } catch {
      return false;
    }
  };

  const onSubmit = async () => {
    try {
      await formRef.current.validate();
    } catch {
      showError(t('请检查输入'));
      return;
    }

    setLoading(true);
    try {
      const res = await API.put('/api/option/', {
        key: 'TimeRatioRules',
        value: rules,
      });
      if (!res.data.success) {
        showError(res.data.message);
        return;
      }
      showSuccess(t('保存成功'));
      refresh?.();
    } catch (error) {
      showError(t('保存失败，请重试'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <Card>
      <Paragraph>
        {t(
          '时间倍率会在请求开始时按规则命中并冻结，后续预扣费、结算和日志都会使用同一个倍率。',
        )}
      </Paragraph>
      <Paragraph>
        <Text type='tertiary'>
          {t(
            '规则按 priority 从高到低匹配；start/end 使用 HH:MM；支持跨午夜；models/groups/user_groups 支持 * 通配。',
          )}
        </Text>
      </Paragraph>
      <Form ref={formRef} values={{ TimeRatioRules: rules }}>
        <Form.TextArea
          field='TimeRatioRules'
          label={t('时间倍率规则')}
          autosize={{ minRows: 16, maxRows: 30 }}
          placeholder={DEFAULT_EXAMPLE}
          rules={[
            {
              validator: (rule, value) => validateRules(value),
              message: t('必须是合法的 JSON 数组'),
            },
          ]}
          onChange={(value) => setRules(value)}
        />
        <Button theme='solid' loading={loading} onClick={onSubmit}>
          {t('保存时间倍率设置')}
        </Button>
      </Form>
    </Card>
  );
}
