const applyNullableProperties = (value) => {
  if (Array.isArray(value)) {
    value.forEach(applyNullableProperties);
    return;
  }

  if (!value || typeof value !== 'object') {
    return;
  }

  if (value['x-nullable'] === true) {
    value.nullable = true;
  }

  Object.values(value).forEach(applyNullableProperties);
};

export default (specification) => {
  applyNullableProperties(specification);
  return specification;
};
