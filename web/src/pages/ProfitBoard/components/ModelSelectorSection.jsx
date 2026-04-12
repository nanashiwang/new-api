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
import React from 'react';
import { Button, Select, Tooltip, Typography } from '@douyinfe/semi-ui';
import { Download } from 'lucide-react';

const { Text } = Typography;

const ModelSelectorSection = ({
  modelNames,
  onModelNamesChange,
  modelNameOptions,
  scopeHasSelection,
  onImportFromScope,
  t,
}) => (
  <div>
    <div className='mb-1.5 flex items-center justify-between gap-2'>
      <Text type='tertiary' size='small'>{t('模型')}</Text>
      <Tooltip
        content={!scopeHasSelection ? t('请先在上方选择范围') : t('从已选的渠道/标签导入全部模型')}
        position='top'
      >
        <Button
          size='small'
          theme='light'
          type='primary'
          icon={<Download size={13} />}
          disabled={!scopeHasSelection}
          onClick={onImportFromScope}
        >
          {t('从已选范围导入')}
        </Button>
      </Tooltip>
    </div>
    <Select
      multiple
      filter
      maxTagCount={5}
      value={modelNames}
      onChange={onModelNamesChange}
      optionList={modelNameOptions}
      placeholder={t('选择模型')}
      emptyContent={t('暂无可用模型')}
      style={{ width: '100%' }}
    />
  </div>
);

export default ModelSelectorSection;
