import assert from 'node:assert/strict';
import {
  extractPlaygroundErrorMessage,
  parsePlaygroundSSEData,
  parsePlaygroundSSEItem,
} from '../src/helpers/playgroundSSE.js';

const chatChunk = parsePlaygroundSSEItem(
  '{"choices":[{"delta":{"content":"ok"}}]}',
);
assert.equal(chatChunk.error, null);
assert.equal(chatChunk.parsed.choices[0].delta.content, 'ok');

const responsesEvent = parsePlaygroundSSEItem(
  'event: response.output_text.delta\ndata: {"type":"response.output_text.delta","delta":"ok"}',
);
assert.equal(responsesEvent.eventType, 'response.output_text.delta');
assert.equal(responsesEvent.parsed.delta, 'ok');

const multilineEvent = parsePlaygroundSSEItem(
  'data: {\ndata:   "value": true\ndata: }',
);
assert.equal(multilineEvent.parsed.value, true);

assert.equal(parsePlaygroundSSEItem('data: [DONE]').isDone, true);
assert.equal(parsePlaygroundSSEData([': ping']).length, 0);
assert.equal(parsePlaygroundSSEData(['data: heartbeat']).length, 0);

const splitEvents = parsePlaygroundSSEData([
  'event: first\ndata: {"index":1}\n\nevent: second\ndata: {"index":2}',
]);
assert.equal(splitEvents.length, 2);
assert.equal(splitEvents[0].parsed.index, 1);
assert.equal(splitEvents[1].parsed.index, 2);

const invalidEvent = parsePlaygroundSSEItem('not-json');
assert.match(invalidEvent.error, /Unexpected token/);

const apiError =
  '{"error":{"message":"分组 vip 下模型 test 无可用渠道 (request id: req-1)","code":"model_not_found"}}';
assert.equal(
  extractPlaygroundErrorMessage(apiError, 'fallback'),
  '分组 vip 下模型 test 无可用渠道 (request id: req-1)',
);
assert.equal(
  extractPlaygroundErrorMessage(`data: ${apiError}`, 'fallback'),
  '分组 vip 下模型 test 无可用渠道 (request id: req-1)',
);
assert.equal(
  extractPlaygroundErrorMessage(
    '{"type":"response.failed","response":{"error":{"message":"upstream failed"}}}',
    'fallback',
  ),
  'upstream failed',
);
assert.equal(
  extractPlaygroundErrorMessage('<html>bad gateway</html>', 'fallback'),
  'fallback',
);
assert.equal(extractPlaygroundErrorMessage('', 'fallback'), 'fallback');

console.log('playground SSE checks passed');
