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
