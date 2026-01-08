'use strict';

const test = require('node:test');
const assert = require('node:assert/strict');

const {
  createHandler,
  RepositoryError,
  ERROR_CODES,
  ERROR_TYPES,
} = require('../../lambda-nodejs/index');

function buildRepository(overrides = {}) {
  return {
    getTableName: () => 'fis-playground-items-dev',
    healthCheck: async () => {},
    createItem: async (item) => {
      if (!item.id) {
        item.id = 'test-id';
      }
    },
    getItem: async (id) => ({
      id,
      name: 'Test Item',
      description: 'Test Description',
      status: 'active',
      created_at: '2024-01-01T00:00:00.000Z',
      updated_at: '2024-01-01T00:00:00.000Z',
    }),
    listItems: async () => ({
      items: [],
      hasMore: false,
    }),
    updateItem: async (id, updates) => ({
      id,
      name: updates.name || 'Updated Item',
      description: updates.description || 'Updated Description',
      status: updates.status || 'active',
      created_at: '2024-01-01T00:00:00.000Z',
      updated_at: '2024-01-02T00:00:00.000Z',
    }),
    deleteItem: async () => {},
    ...overrides,
  };
}

test('GET /health returns healthy response', async () => {
  const handler = createHandler({ repository: buildRepository() });
  const response = await handler({ httpMethod: 'GET', path: '/health' });
  const body = JSON.parse(response.body);

  assert.equal(response.statusCode, 200);
  assert.equal(body.success, true);
  assert.equal(body.data.status, 'healthy');
});

test('POST /items validates payload', async () => {
  const handler = createHandler({ repository: buildRepository() });
  const response = await handler({
    httpMethod: 'POST',
    path: '/items',
    body: JSON.stringify({ name: '', description: 'Test Description' }),
  });
  const body = JSON.parse(response.body);

  assert.equal(response.statusCode, 400);
  assert.equal(body.error.code, ERROR_CODES.MISSING_FIELD);
  assert.equal(body.error.type, ERROR_TYPES.VALIDATION);
});

test('POST /items creates item with id', async () => {
  const handler = createHandler({ repository: buildRepository() });
  const response = await handler({
    httpMethod: 'POST',
    path: '/items',
    body: JSON.stringify({ name: 'Test Item', description: 'Test Description' }),
  });
  const body = JSON.parse(response.body);

  assert.equal(response.statusCode, 201);
  assert.equal(body.success, true);
  assert.ok(body.data.id);
  assert.equal(body.data.status, 'active');
});

test('GET /items validates limit', async () => {
  const handler = createHandler({ repository: buildRepository() });
  const response = await handler({
    httpMethod: 'GET',
    path: '/items',
    queryStringParameters: { limit: 'invalid' },
  });
  const body = JSON.parse(response.body);

  assert.equal(response.statusCode, 400);
  assert.equal(body.error.code, ERROR_CODES.INVALID_FORMAT);
});

test('PUT /items/{id} validates status', async () => {
  const handler = createHandler({ repository: buildRepository() });
  const response = await handler({
    httpMethod: 'PUT',
    path: '/items/test-id',
    body: JSON.stringify({ status: 'bogus' }),
  });
  const body = JSON.parse(response.body);

  assert.equal(response.statusCode, 400);
  assert.equal(body.error.code, ERROR_CODES.INVALID_VALUE);
});

test('DELETE /items/{id} maps not found errors', async () => {
  const handler = createHandler({
    repository: buildRepository({
      deleteItem: async () => {
        throw new RepositoryError('not_found', 'item not found');
      },
    }),
  });
  const response = await handler({
    httpMethod: 'DELETE',
    path: '/items/missing',
  });
  const body = JSON.parse(response.body);

  assert.equal(response.statusCode, 404);
  assert.equal(body.error.type, ERROR_TYPES.NOT_FOUND);
});
