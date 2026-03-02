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

import React, { useRef, useState } from 'react';
import { Form, Button, SideSheet, Space, Divider, Typography } from '@douyinfe/semi-ui';
import { IconSearch, IconFilter } from '@douyinfe/semi-icons';

const TokensFilters = ({
  formInitValues,
  setFormApi,
  searchTokens,
  groupOptions,
  loading,
  searching,
  t,
}) => {
  // Handle form reset and immediate search
  const formApiRef = useRef(null);
  const [advancedVisible, setAdvancedVisible] = useState(false);
  const [activeAdvancedCount, setActiveAdvancedCount] = useState(0);
  const { Text } = Typography;

  const getAdvancedCount = () => {
    if (!formApiRef.current) return 0;
    const values = formApiRef.current.getValues() || {};
    const fields = [
      values.searchBalanceMin,
      values.searchBalanceMax,
      values.searchUsedBalanceMin,
      values.searchUsedBalanceMax,
      values.searchAmountSort,
    ];
    return fields.filter(
      (value) => value !== '' && value !== null && value !== undefined,
    ).length;
  };

  const refreshAdvancedCount = () => {
    setActiveAdvancedCount(getAdvancedCount());
  };

  const handleReset = () => {
    if (!formApiRef.current) return;
    formApiRef.current.reset();
    setActiveAdvancedCount(0);
    setTimeout(() => {
      searchTokens();
    }, 100);
  };

  // 保留“默认 + 金额升降序”三档，满足排序需求同时避免筛选区堆叠太多控件。
  const amountSortOptions = [
    { label: t('默认排序'), value: '' },
    { label: t('金额降序'), value: 'quota_desc' },
    { label: t('金额升序'), value: 'quota_asc' },
  ];

  const openAdvancedFilters = () => {
    refreshAdvancedCount();
    setAdvancedVisible(true);
  };

  const applyAdvancedFilters = () => {
    refreshAdvancedCount();
    setAdvancedVisible(false);
    setTimeout(() => {
      searchTokens(1);
    }, 100);
  };

  const clearAdvancedFilters = () => {
    if (!formApiRef.current) return;
    // 只清空金额相关字段，不影响关键词输入。
    formApiRef.current.setValues({
      searchBalanceMin: '',
      searchBalanceMax: '',
      searchUsedBalanceMin: '',
      searchUsedBalanceMax: '',
      searchAmountSort: '',
    });
    setActiveAdvancedCount(0);
  };

  return (
    <Form
      initValues={formInitValues}
      getFormApi={(api) => {
        setFormApi(api);
        formApiRef.current = api;
      }}
      onSubmit={() => searchTokens(1)}
      allowEmpty={true}
      autoComplete='off'
      layout='horizontal'
      trigger='change'
      stopValidateWithError={false}
      className='w-full md:w-auto order-1 md:order-2'
    >
      <div className='flex flex-col md:flex-row items-center gap-2 w-full md:w-auto'>
        <div className='relative w-full md:w-48'>
          <Form.Input
            field='searchKeyword'
            prefix={<IconSearch />}
            placeholder={t('搜索关键字')}
            showClear
            pure
            size='small'
          />
        </div>

        <div className='relative w-full md:w-48'>
          <Form.Input
            field='searchToken'
            prefix={<IconSearch />}
            placeholder={t('密钥')}
            showClear
            pure
            size='small'
          />
        </div>

        <div className='w-full md:w-64'>
          <Form.Select
            field='searchGroup'
            placeholder={t('选择分组')}
            optionList={groupOptions}
            className='w-full'
            showClear
            pure
            size='small'
            onChange={() => {
              // 分组属于高频过滤条件，切换后立即刷新结果，减少一次额外点击。
              setTimeout(() => {
                searchTokens(1);
              }, 100);
            }}
          />
        </div>

        <div className='flex gap-2 w-full md:w-auto'>
          <Button
            type='tertiary'
            htmlType='submit'
            loading={loading || searching}
            className='flex-1 md:flex-initial md:w-auto'
            size='small'
          >
            {t('查询')}
          </Button>
          <Button
            type='tertiary'
            onClick={openAdvancedFilters}
            className='flex-1 md:flex-initial md:w-auto'
            icon={<IconFilter />}
            size='small'
          >
            {activeAdvancedCount > 0
              ? `${t('高级筛选')}(${activeAdvancedCount})`
              : t('高级筛选')}
          </Button>

          <Button
            type='tertiary'
            onClick={handleReset}
            className='flex-1 md:flex-initial md:w-auto'
            size='small'
          >
            {t('重置')}
          </Button>
        </div>
      </div>

      <SideSheet
        visible={advancedVisible}
        onCancel={() => setAdvancedVisible(false)}
        title={t('高级筛选')}
        placement='right'
        width={420}
        bodyStyle={{ padding: 16 }}
        footer={
          <div className='flex justify-end'>
            <Space>
              <Button type='tertiary' onClick={clearAdvancedFilters}>
                {t('清空当前设置')}
              </Button>
              <Button type='primary' theme='solid' onClick={applyAdvancedFilters}>
                {t('应用筛选')}
              </Button>
            </Space>
          </div>
        }
      >
        <div className='grid grid-cols-1 gap-3'>
          <Text type='tertiary' size='small'>
            {t('额度筛选')}
          </Text>
          <Form.InputNumber
            field='searchBalanceMin'
            placeholder={t('额度最小值')}
            noLabel
            min={0}
            precision={0}
            hideButtons
            onChange={refreshAdvancedCount}
          />
          <Form.InputNumber
            field='searchBalanceMax'
            placeholder={t('额度最大值')}
            noLabel
            min={0}
            precision={0}
            hideButtons
            onChange={refreshAdvancedCount}
          />
          <Divider margin='4px' />
          <Text type='tertiary' size='small'>
            {t('已使用余额筛选')}
          </Text>
          <Form.InputNumber
            field='searchUsedBalanceMin'
            placeholder={t('已使用余额最小值')}
            noLabel
            min={0}
            precision={0}
            hideButtons
            onChange={refreshAdvancedCount}
          />
          <Form.InputNumber
            field='searchUsedBalanceMax'
            placeholder={t('已使用余额最大值')}
            noLabel
            min={0}
            precision={0}
            hideButtons
            onChange={refreshAdvancedCount}
          />
          <Divider margin='4px' />
          <Text type='tertiary' size='small'>
            {t('金额排序')}
          </Text>
          <Form.Select
            field='searchAmountSort'
            placeholder={t('金额排序')}
            optionList={amountSortOptions}
            noLabel
            showClear
            onChange={refreshAdvancedCount}
          />
        </div>
      </SideSheet>
    </Form>
  );
};

export default TokensFilters;
