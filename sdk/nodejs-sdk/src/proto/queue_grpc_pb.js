// GENERATED CODE -- DO NOT EDIT!

'use strict';
var grpc = require('@grpc/grpc-js');
var queue_pb = require('./queue_pb.js');

function serialize_queue_BulkCreateQueueRequest(arg) {
  if (!(arg instanceof queue_pb.BulkCreateQueueRequest)) {
    throw new Error('Expected argument of type queue.BulkCreateQueueRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_queue_BulkCreateQueueRequest(buffer_arg) {
  return queue_pb.BulkCreateQueueRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_queue_BulkCreateQueueResponse(arg) {
  if (!(arg instanceof queue_pb.BulkCreateQueueResponse)) {
    throw new Error('Expected argument of type queue.BulkCreateQueueResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_queue_BulkCreateQueueResponse(buffer_arg) {
  return queue_pb.BulkCreateQueueResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_queue_CreateQueueRequest(arg) {
  if (!(arg instanceof queue_pb.CreateQueueRequest)) {
    throw new Error('Expected argument of type queue.CreateQueueRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_queue_CreateQueueRequest(buffer_arg) {
  return queue_pb.CreateQueueRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_queue_CreateQueueResponse(arg) {
  if (!(arg instanceof queue_pb.CreateQueueResponse)) {
    throw new Error('Expected argument of type queue.CreateQueueResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_queue_CreateQueueResponse(buffer_arg) {
  return queue_pb.CreateQueueResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_queue_DeleteQueueRequest(arg) {
  if (!(arg instanceof queue_pb.DeleteQueueRequest)) {
    throw new Error('Expected argument of type queue.DeleteQueueRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_queue_DeleteQueueRequest(buffer_arg) {
  return queue_pb.DeleteQueueRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_queue_DeleteQueueResponse(arg) {
  if (!(arg instanceof queue_pb.DeleteQueueResponse)) {
    throw new Error('Expected argument of type queue.DeleteQueueResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_queue_DeleteQueueResponse(buffer_arg) {
  return queue_pb.DeleteQueueResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_queue_EnqueueMessageRequest(arg) {
  if (!(arg instanceof queue_pb.EnqueueMessageRequest)) {
    throw new Error('Expected argument of type queue.EnqueueMessageRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_queue_EnqueueMessageRequest(buffer_arg) {
  return queue_pb.EnqueueMessageRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_queue_EnqueueMessageResponse(arg) {
  if (!(arg instanceof queue_pb.EnqueueMessageResponse)) {
    throw new Error('Expected argument of type queue.EnqueueMessageResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_queue_EnqueueMessageResponse(buffer_arg) {
  return queue_pb.EnqueueMessageResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_queue_GetQueueRequest(arg) {
  if (!(arg instanceof queue_pb.GetQueueRequest)) {
    throw new Error('Expected argument of type queue.GetQueueRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_queue_GetQueueRequest(buffer_arg) {
  return queue_pb.GetQueueRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_queue_GetQueueResponse(arg) {
  if (!(arg instanceof queue_pb.GetQueueResponse)) {
    throw new Error('Expected argument of type queue.GetQueueResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_queue_GetQueueResponse(buffer_arg) {
  return queue_pb.GetQueueResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_queue_GetQueuesRequest(arg) {
  if (!(arg instanceof queue_pb.GetQueuesRequest)) {
    throw new Error('Expected argument of type queue.GetQueuesRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_queue_GetQueuesRequest(buffer_arg) {
  return queue_pb.GetQueuesRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_queue_GetQueuesResponse(arg) {
  if (!(arg instanceof queue_pb.GetQueuesResponse)) {
    throw new Error('Expected argument of type queue.GetQueuesResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_queue_GetQueuesResponse(buffer_arg) {
  return queue_pb.GetQueuesResponse.deserializeBinary(new Uint8Array(buffer_arg));
}


var QueueServiceService = exports.QueueServiceService = {
  createQueue: {
    path: '/queue.QueueService/CreateQueue',
    requestStream: false,
    responseStream: false,
    requestType: queue_pb.CreateQueueRequest,
    responseType: queue_pb.CreateQueueResponse,
    requestSerialize: serialize_queue_CreateQueueRequest,
    requestDeserialize: deserialize_queue_CreateQueueRequest,
    responseSerialize: serialize_queue_CreateQueueResponse,
    responseDeserialize: deserialize_queue_CreateQueueResponse,
  },
  bulkCreateQueue: {
    path: '/queue.QueueService/BulkCreateQueue',
    requestStream: false,
    responseStream: false,
    requestType: queue_pb.BulkCreateQueueRequest,
    responseType: queue_pb.BulkCreateQueueResponse,
    requestSerialize: serialize_queue_BulkCreateQueueRequest,
    requestDeserialize: deserialize_queue_BulkCreateQueueRequest,
    responseSerialize: serialize_queue_BulkCreateQueueResponse,
    responseDeserialize: deserialize_queue_BulkCreateQueueResponse,
  },
  getQueue: {
    path: '/queue.QueueService/GetQueue',
    requestStream: false,
    responseStream: false,
    requestType: queue_pb.GetQueueRequest,
    responseType: queue_pb.GetQueueResponse,
    requestSerialize: serialize_queue_GetQueueRequest,
    requestDeserialize: deserialize_queue_GetQueueRequest,
    responseSerialize: serialize_queue_GetQueueResponse,
    responseDeserialize: deserialize_queue_GetQueueResponse,
  },
  getQueues: {
    path: '/queue.QueueService/GetQueues',
    requestStream: false,
    responseStream: false,
    requestType: queue_pb.GetQueuesRequest,
    responseType: queue_pb.GetQueuesResponse,
    requestSerialize: serialize_queue_GetQueuesRequest,
    requestDeserialize: deserialize_queue_GetQueuesRequest,
    responseSerialize: serialize_queue_GetQueuesResponse,
    responseDeserialize: deserialize_queue_GetQueuesResponse,
  },
  deleteQueue: {
    path: '/queue.QueueService/DeleteQueue',
    requestStream: false,
    responseStream: false,
    requestType: queue_pb.DeleteQueueRequest,
    responseType: queue_pb.DeleteQueueResponse,
    requestSerialize: serialize_queue_DeleteQueueRequest,
    requestDeserialize: deserialize_queue_DeleteQueueRequest,
    responseSerialize: serialize_queue_DeleteQueueResponse,
    responseDeserialize: deserialize_queue_DeleteQueueResponse,
  },
  enqueueMessage: {
    path: '/queue.QueueService/EnqueueMessage',
    requestStream: false,
    responseStream: false,
    requestType: queue_pb.EnqueueMessageRequest,
    responseType: queue_pb.EnqueueMessageResponse,
    requestSerialize: serialize_queue_EnqueueMessageRequest,
    requestDeserialize: deserialize_queue_EnqueueMessageRequest,
    responseSerialize: serialize_queue_EnqueueMessageResponse,
    responseDeserialize: deserialize_queue_EnqueueMessageResponse,
  },
};

exports.QueueServiceClient = grpc.makeGenericClientConstructor(QueueServiceService, 'QueueService');
