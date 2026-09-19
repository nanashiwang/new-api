// Provider-declared names are diagnostics, never proof of the underlying model.
export function getResponseModelInfo(other, requestedModel) {
  const observation = other?.response_model;
  if (
    !observation ||
    typeof observation.returned_model !== 'string' ||
    !observation.returned_model.trim()
  )
    return null;
  const text = (value, fallback) =>
    typeof value === 'string' && value ? value : fallback;
  return {
    requested: text(observation.requested_model, requestedModel),
    upstream: text(
      observation.upstream_model,
      other?.upstream_model_name || requestedModel,
    ),
    returned: observation.returned_model,
    mismatch: observation.mismatch === true,
  };
}
