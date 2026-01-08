'use strict';

const crypto = require('node:crypto');
let cachedDynamoModules;

function loadDynamoModules() {
  if (!cachedDynamoModules) {
    // Lazy-load so local tests don't require SDK modules to be installed.
    const { DynamoDBClient, DescribeTableCommand } = require('@aws-sdk/client-dynamodb');
    const {
      DynamoDBDocumentClient,
      PutCommand,
      GetCommand,
      ScanCommand,
      UpdateCommand,
      DeleteCommand,
    } = require('@aws-sdk/lib-dynamodb');

    cachedDynamoModules = {
      DynamoDBClient,
      DynamoDBDocumentClient,
      DescribeTableCommand,
      PutCommand,
      GetCommand,
      ScanCommand,
      UpdateCommand,
      DeleteCommand,
    };
  }
  return cachedDynamoModules;
}

const ERROR_TYPES = {
  VALIDATION: 'validation',
  NOT_FOUND: 'not_found',
  CONFLICT: 'conflict',
  DATABASE: 'database',
  SYSTEM: 'system',
  AUTH: 'authentication',
  RATE: 'rate_limit',
};

const ERROR_CODES = {
  INVALID_REQUEST: 'INVALID_REQUEST',
  MISSING_FIELD: 'MISSING_FIELD',
  INVALID_FORMAT: 'INVALID_FORMAT',
  VALUE_TOO_LONG: 'VALUE_TOO_LONG',
  INVALID_VALUE: 'INVALID_VALUE',
  NOT_FOUND: 'NOT_FOUND',
  ALREADY_EXISTS: 'ALREADY_EXISTS',
  DATABASE_ERROR: 'DATABASE_ERROR',
  CONNECTION_ERROR: 'CONNECTION_ERROR',
  OPERATION_FAILED: 'OPERATION_FAILED',
  THROUGHPUT_EXCEEDED: 'THROUGHPUT_EXCEEDED',
  INTERNAL_ERROR: 'INTERNAL_ERROR',
  SERVICE_UNAVAILABLE: 'SERVICE_UNAVAILABLE',
  TIMEOUT: 'TIMEOUT',
  UNAUTHORIZED: 'UNAUTHORIZED',
  FORBIDDEN: 'FORBIDDEN',
  RATE_LIMIT_EXCEEDED: 'RATE_LIMIT_EXCEEDED',
};

const DEFAULT_HEADERS = {
  'Content-Type': 'application/json',
  'Access-Control-Allow-Origin': '*',
  'Access-Control-Allow-Headers': 'Content-Type,Authorization',
  'Access-Control-Allow-Methods': 'GET,POST,PUT,DELETE,OPTIONS',
};

const VALID_STATUSES = new Set(['active', 'inactive', 'pending']);

class RepositoryError extends Error {
  constructor(kind, message, cause) {
    super(message);
    this.kind = kind;
    this.cause = cause;
  }
}

function normalizePath(path) {
  if (!path) {
    return '/';
  }
  if (path.length > 1 && path.endsWith('/')) {
    return path.slice(0, -1);
  }
  return path;
}

function jsonResponse(statusCode, payload) {
  return {
    statusCode,
    headers: DEFAULT_HEADERS,
    body: JSON.stringify(payload),
  };
}

function buildErrorInfo(apiError) {
  const errorInfo = {
    code: apiError.code,
    message: apiError.message,
    type: apiError.type,
  };
  if (apiError.details) {
    errorInfo.details = apiError.details;
  }
  return errorInfo;
}

function errorResponse(apiError) {
  if (apiError.cause) {
    console.error(
      `API Error ${apiError.code}: ${apiError.message}`,
      apiError.cause
    );
  } else {
    console.error(`API Error ${apiError.code}: ${apiError.message}`);
  }

  return jsonResponse(apiError.statusCode, {
    success: false,
    error: buildErrorInfo(apiError),
  });
}

function validationErrorResponse(code, message, details) {
  return errorResponse({
    type: ERROR_TYPES.VALIDATION,
    code,
    message,
    details,
    statusCode: 400,
  });
}

function missingParameterError(parameter) {
  return errorResponse({
    type: ERROR_TYPES.VALIDATION,
    code: ERROR_CODES.MISSING_FIELD,
    message: `${parameter} is required`,
    details: `The required parameter '${parameter}' is missing from the request`,
    statusCode: 400,
  });
}

function jsonParseError(err) {
  return errorResponse({
    type: ERROR_TYPES.VALIDATION,
    code: ERROR_CODES.INVALID_FORMAT,
    message: 'Invalid JSON format',
    details: 'Request body contains malformed JSON',
    statusCode: 400,
    cause: err,
  });
}

function mapValidationError(err) {
  const message = err && err.message ? err.message : 'Invalid request';
  let code = ERROR_CODES.INVALID_REQUEST;
  const normalized = message.toLowerCase();

  if (containsAny(normalized, ['empty', 'required'])) {
    code = ERROR_CODES.MISSING_FIELD;
  } else if (containsAny(normalized, ['too long', 'exceed'])) {
    code = ERROR_CODES.VALUE_TOO_LONG;
  } else if (containsAny(normalized, ['invalid', 'format'])) {
    code = ERROR_CODES.INVALID_FORMAT;
  } else if (containsAny(normalized, ['status', 'one of'])) {
    code = ERROR_CODES.INVALID_VALUE;
  }

  return {
    type: ERROR_TYPES.VALIDATION,
    code,
    message,
    statusCode: 400,
    cause: err,
  };
}

function mapRepositoryError(err) {
  if (!err) {
    return null;
  }

  if (err instanceof RepositoryError) {
    switch (err.kind) {
      case 'not_found':
        return {
          type: ERROR_TYPES.NOT_FOUND,
          code: ERROR_CODES.NOT_FOUND,
          message: 'Resource not found',
          details: err.message,
          statusCode: 404,
          cause: err.cause || err,
        };
      case 'conflict':
        return {
          type: ERROR_TYPES.CONFLICT,
          code: ERROR_CODES.ALREADY_EXISTS,
          message: 'Resource already exists',
          details: err.message,
          statusCode: 409,
          cause: err.cause || err,
        };
      case 'invalid_input':
        return {
          type: ERROR_TYPES.VALIDATION,
          code: ERROR_CODES.INVALID_REQUEST,
          message: 'Invalid input provided',
          details: err.message,
          statusCode: 400,
          cause: err.cause || err,
        };
      case 'connection_failed':
        return {
          type: ERROR_TYPES.DATABASE,
          code: ERROR_CODES.CONNECTION_ERROR,
          message: 'Database connection failed',
          details: 'Unable to connect to the database',
          statusCode: 503,
          cause: err.cause || err,
        };
      case 'operation_failed': {
        if (containsAny(err.message.toLowerCase(), [
          'throughput',
          'capacity',
          'throttling',
          'rate',
        ])) {
          return {
            type: ERROR_TYPES.DATABASE,
            code: ERROR_CODES.THROUGHPUT_EXCEEDED,
            message: 'Database throughput exceeded',
            details: 'Please retry your request after a brief delay',
            statusCode: 429,
            cause: err.cause || err,
          };
        }
        return {
          type: ERROR_TYPES.DATABASE,
          code: ERROR_CODES.DATABASE_ERROR,
          message: 'Database operation failed',
          details: err.message,
          statusCode: 500,
          cause: err.cause || err,
        };
      }
      default:
        break;
    }
  }

  return {
    type: ERROR_TYPES.SYSTEM,
    code: ERROR_CODES.INTERNAL_ERROR,
    message: 'An unexpected error occurred',
    details: 'Please try again later',
    statusCode: 500,
    cause: err,
  };
}

function containsAny(message, keywords) {
  return keywords.some((keyword) => message.includes(keyword));
}

function parseJsonBody(event) {
  if (!event || event.body == null) {
    throw new Error('empty request body');
  }
  const body = event.isBase64Encoded
    ? Buffer.from(event.body, 'base64').toString('utf8')
    : event.body;
  return JSON.parse(body);
}

function validateCreateItemRequest(request) {
  if (!request || typeof request !== 'object') {
    throw new Error('invalid request body');
  }

  const name = request.name;
  const description = request.description;

  if (typeof name !== 'string' || name.trim() === '') {
    throw new Error('name cannot be empty');
  }
  if (name.length > 100) {
    throw new Error('name cannot exceed 100 characters');
  }
  if (typeof description !== 'string' || description.trim() === '') {
    throw new Error('description cannot be empty');
  }
  if (description.length > 500) {
    throw new Error('description cannot exceed 500 characters');
  }
}

function validateUpdateItemRequest(request) {
  if (!request || typeof request !== 'object') {
    throw new Error('invalid request body');
  }

  if (Object.prototype.hasOwnProperty.call(request, 'name')) {
    const name = request.name;
    if (typeof name !== 'string' || name.trim() === '') {
      throw new Error('name cannot be empty');
    }
    if (name.length > 100) {
      throw new Error('name cannot exceed 100 characters');
    }
  }

  if (Object.prototype.hasOwnProperty.call(request, 'description')) {
    const description = request.description;
    if (typeof description !== 'string' || description.trim() === '') {
      throw new Error('description cannot be empty');
    }
    if (description.length > 500) {
      throw new Error('description cannot exceed 500 characters');
    }
  }

  if (Object.prototype.hasOwnProperty.call(request, 'status')) {
    const status = request.status;
    if (typeof status !== 'string' || !VALID_STATUSES.has(status)) {
      throw new Error('status must be one of: active, inactive, pending');
    }
  }
}

function createNewItem(name, description, now) {
  const timestamp = now.toISOString();
  return {
    id: '',
    name,
    description,
    status: 'active',
    created_at: timestamp,
    updated_at: timestamp,
  };
}

function buildUpdateExpression(updates, timestamp) {
  const updateParts = ['updated_at = :updated_at'];
  const expressionAttributeValues = {
    ':updated_at': timestamp,
  };
  const expressionAttributeNames = {};

  if (Object.prototype.hasOwnProperty.call(updates, 'name')) {
    updateParts.push('#name = :name');
    expressionAttributeValues[':name'] = updates.name;
    expressionAttributeNames['#name'] = 'name';
  }

  if (Object.prototype.hasOwnProperty.call(updates, 'description')) {
    updateParts.push('description = :description');
    expressionAttributeValues[':description'] = updates.description;
  }

  if (Object.prototype.hasOwnProperty.call(updates, 'status')) {
    updateParts.push('#status = :status');
    expressionAttributeValues[':status'] = updates.status;
    expressionAttributeNames['#status'] = 'status';
  }

  return {
    updateExpression: `SET ${updateParts.join(', ')}`,
    expressionAttributeValues,
    expressionAttributeNames,
  };
}

function createDynamoRepository(config) {
  const tableName = config.tableName;
  const modules = loadDynamoModules();
  const dynamoClient = config.dynamoClient || new modules.DynamoDBClient({});
  const documentClient =
    config.documentClient || modules.DynamoDBDocumentClient.from(dynamoClient);

  if (!tableName) {
    throw new Error('DYNAMODB_TABLE_NAME environment variable is required');
  }

  return {
    getTableName() {
      return tableName;
    },
    async healthCheck() {
      await dynamoClient.send(
        new modules.DescribeTableCommand({ TableName: tableName })
      );
    },
    async createItem(item) {
      if (!item.id) {
        item.id = crypto.randomUUID();
      }

      const params = {
        TableName: tableName,
        Item: item,
        ConditionExpression: 'attribute_not_exists(id)',
      };

      try {
        await documentClient.send(new modules.PutCommand(params));
      } catch (err) {
        throw mapDynamoError(err, { operation: 'create' });
      }
    },
    async getItem(id) {
      if (!id) {
        throw new RepositoryError('invalid_input', 'item ID cannot be empty');
      }

      try {
        const result = await documentClient.send(
          new modules.GetCommand({
            TableName: tableName,
            Key: { id },
          })
        );
        if (!result.Item) {
          throw new RepositoryError('not_found', 'item not found');
        }
        return result.Item;
      } catch (err) {
        if (err instanceof RepositoryError) {
          throw err;
        }
        throw mapDynamoError(err, { operation: 'get' });
      }
    },
    async listItems(options) {
      const limit = options && options.limit ? options.limit : 50;
      const params = {
        TableName: tableName,
        Limit: limit,
      };

      try {
        const result = await documentClient.send(new modules.ScanCommand(params));
        return {
          items: result.Items || [],
          hasMore: Boolean(result.LastEvaluatedKey),
        };
      } catch (err) {
        throw mapDynamoError(err, { operation: 'list' });
      }
    },
    async updateItem(id, updates) {
      if (!id) {
        throw new RepositoryError('invalid_input', 'item ID cannot be empty');
      }
      if (!updates || typeof updates !== 'object') {
        throw new RepositoryError('invalid_input', 'updates cannot be nil');
      }

      const timestamp = new Date().toISOString();
      const expression = buildUpdateExpression(updates, timestamp);

      const params = {
        TableName: tableName,
        Key: { id },
        UpdateExpression: expression.updateExpression,
        ExpressionAttributeValues: expression.expressionAttributeValues,
        ConditionExpression: 'attribute_exists(id)',
        ReturnValues: 'ALL_NEW',
      };

      if (Object.keys(expression.expressionAttributeNames).length > 0) {
        params.ExpressionAttributeNames = expression.expressionAttributeNames;
      }

      try {
        const result = await documentClient.send(new modules.UpdateCommand(params));
        return result.Attributes;
      } catch (err) {
        throw mapDynamoError(err, { operation: 'update' });
      }
    },
    async deleteItem(id) {
      if (!id) {
        throw new RepositoryError('invalid_input', 'item ID cannot be empty');
      }

      const params = {
        TableName: tableName,
        Key: { id },
        ConditionExpression: 'attribute_exists(id)',
      };

      try {
        await documentClient.send(new modules.DeleteCommand(params));
      } catch (err) {
        throw mapDynamoError(err, { operation: 'delete' });
      }
    },
  };
}

function mapDynamoError(err, context) {
  if (!err) {
    return new RepositoryError('operation_failed', 'unknown error');
  }

  const code = err.name || err.code;

  if (code === 'ConditionalCheckFailedException') {
    if (context.operation === 'create') {
      return new RepositoryError('conflict', 'item with this ID already exists', err);
    }
    return new RepositoryError('not_found', 'item not found', err);
  }

  if (code === 'ResourceNotFoundException') {
    return new RepositoryError('not_found', err.message || 'resource not found', err);
  }

  if (code === 'ProvisionedThroughputExceededException') {
    return new RepositoryError('operation_failed', 'throughput exceeded', err);
  }

  if (code === 'ResourceInUseException' || code === 'InternalServerError') {
    return new RepositoryError('operation_failed', err.message || 'operation failed', err);
  }

  if (code === 'ValidationException') {
    return new RepositoryError('invalid_input', err.message || 'invalid input', err);
  }

  if (code === 'NetworkingError' || code === 'TimeoutError') {
    return new RepositoryError('connection_failed', err.message || 'connection failed', err);
  }

  return new RepositoryError('operation_failed', err.message || 'operation failed', err);
}

function createHandler(options) {
  const repository = options.repository;
  const getRepository = options.getRepository;
  const now = options.now || (() => new Date());

  function repoProvider() {
    if (repository) {
      return repository;
    }
    if (getRepository) {
      return getRepository();
    }
    throw new Error('repository not configured');
  }

  return async function handler(event) {
    const method = (event && event.httpMethod ? event.httpMethod : '').toUpperCase();
    const path = normalizePath(event && event.path ? event.path : '/');
    const query = (event && event.queryStringParameters) || {};

    try {
      if (method === 'GET' && (path === '/' || path === '/health')) {
        return jsonResponse(200, {
          success: true,
          data: {
            status: 'healthy',
            service: 'FIS Playground API',
          },
        });
      }

      if (method === 'GET' && path === '/health/db') {
        let status = 'healthy';
        let message = 'Connected';
        let tableName = 'unknown';

        try {
          const repo = repoProvider();
          tableName = repo.getTableName ? repo.getTableName() : 'unknown';
          await repo.healthCheck();
        } catch (err) {
          status = 'unhealthy';
          message = err.message || 'DynamoDB health check failed';
        }

        const success = status === 'healthy';
        return jsonResponse(success ? 200 : 503, {
          success,
          data: {
            status,
            service: 'DynamoDB',
            message,
            tableName,
          },
        });
      }

      if (path === '/items') {
        if (method === 'GET') {
          let limit = 50;
          if (query.limit) {
            const parsed = Number.parseInt(query.limit, 10);
            if (Number.isNaN(parsed)) {
              return validationErrorResponse(
                ERROR_CODES.INVALID_FORMAT,
                'Invalid limit parameter',
                'Limit must be a valid integer'
              );
            }
            if (parsed <= 0 || parsed > 100) {
              return validationErrorResponse(
                ERROR_CODES.INVALID_VALUE,
                'Invalid limit value',
                'Limit must be between 1 and 100'
              );
            }
            limit = parsed;
          }

          const repo = repoProvider();
          const result = await repo.listItems({ limit });
          const responseData = {
            items: result.items,
            has_more: result.hasMore,
            count: result.items.length,
          };

          if (result.hasMore) {
            responseData.next_token = 'pagination_token_placeholder';
          }

          return jsonResponse(200, { success: true, data: responseData });
        }

        if (method === 'POST') {
          let payload;
          try {
            payload = parseJsonBody(event);
          } catch (err) {
            return jsonParseError(err);
          }

          try {
            validateCreateItemRequest(payload);
          } catch (err) {
            return errorResponse(mapValidationError(err));
          }

          const item = createNewItem(payload.name, payload.description, now());
          const repo = repoProvider();
          await repo.createItem(item);

          return jsonResponse(201, { success: true, data: item });
        }
      }

      if (path.startsWith('/items/')) {
        const parts = path.split('/');
        const itemId = parts.length > 2 ? parts[2] : '';

        if (!itemId) {
          return missingParameterError('Item ID');
        }

        if (method === 'GET') {
          const repo = repoProvider();
          const item = await repo.getItem(itemId);
          return jsonResponse(200, { success: true, data: item });
        }

        if (method === 'PUT') {
          let payload;
          try {
            payload = parseJsonBody(event);
          } catch (err) {
            return jsonParseError(err);
          }

          try {
            validateUpdateItemRequest(payload);
          } catch (err) {
            return errorResponse(mapValidationError(err));
          }

          const repo = repoProvider();
          const item = await repo.updateItem(itemId, payload);
          return jsonResponse(200, { success: true, data: item });
        }

        if (method === 'DELETE') {
          const repo = repoProvider();
          await repo.deleteItem(itemId);
          return jsonResponse(200, {
            success: true,
            data: {
              message: 'Item deleted successfully',
              id: itemId,
            },
          });
        }
      }

      return errorResponse({
        type: ERROR_TYPES.NOT_FOUND,
        code: ERROR_CODES.NOT_FOUND,
        message: 'Resource not found',
        details: 'Route not found',
        statusCode: 404,
      });
    } catch (err) {
      const apiErr = mapRepositoryError(err);
      return errorResponse(apiErr);
    }
  };
}

let cachedRepository;
function getRepository() {
  if (!cachedRepository) {
    cachedRepository = createDynamoRepository({
      tableName: process.env.DYNAMODB_TABLE_NAME,
    });
  }
  return cachedRepository;
}

const handler = createHandler({ getRepository });

module.exports = {
  handler,
  createHandler,
  createDynamoRepository,
  RepositoryError,
  ERROR_CODES,
  ERROR_TYPES,
};
