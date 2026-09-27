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
import { expressionDiff } from '../../../../helpers/expressionDiff';

export default function ExpressionDiff({ value, baseline }) {
  return (
    <code
      className='block text-xs leading-relaxed whitespace-pre-wrap break-all'
      style={{ overflowWrap: 'anywhere' }}
    >
      {expressionDiff(value, baseline).map((part, index) =>
        part.changed ? (
          <mark key={index} className='rounded-sm bg-amber-500/25 text-inherit'>
            {part.text}
          </mark>
        ) : (
          <React.Fragment key={index}>{part.text}</React.Fragment>
        ),
      )}
    </code>
  );
}
