// GENERATED CODE -- DO NOT EDIT!

'use strict';
var grpc = require('@grpc/grpc-js');
var scheduled_job_pb = require('./scheduled_job_pb.js');

function serialize_scheduledjob_CreateOneOffScheduledJobRequest(arg) {
  if (!(arg instanceof scheduled_job_pb.CreateOneOffScheduledJobRequest)) {
    throw new Error('Expected argument of type scheduledjob.CreateOneOffScheduledJobRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_scheduledjob_CreateOneOffScheduledJobRequest(buffer_arg) {
  return scheduled_job_pb.CreateOneOffScheduledJobRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_scheduledjob_CreateRecurringScheduledJobRequest(arg) {
  if (!(arg instanceof scheduled_job_pb.CreateRecurringScheduledJobRequest)) {
    throw new Error('Expected argument of type scheduledjob.CreateRecurringScheduledJobRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_scheduledjob_CreateRecurringScheduledJobRequest(buffer_arg) {
  return scheduled_job_pb.CreateRecurringScheduledJobRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_scheduledjob_CreateScheduledJobResponse(arg) {
  if (!(arg instanceof scheduled_job_pb.CreateScheduledJobResponse)) {
    throw new Error('Expected argument of type scheduledjob.CreateScheduledJobResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_scheduledjob_CreateScheduledJobResponse(buffer_arg) {
  return scheduled_job_pb.CreateScheduledJobResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_scheduledjob_DeleteScheduledJobRequest(arg) {
  if (!(arg instanceof scheduled_job_pb.DeleteScheduledJobRequest)) {
    throw new Error('Expected argument of type scheduledjob.DeleteScheduledJobRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_scheduledjob_DeleteScheduledJobRequest(buffer_arg) {
  return scheduled_job_pb.DeleteScheduledJobRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_scheduledjob_DeleteScheduledJobResponse(arg) {
  if (!(arg instanceof scheduled_job_pb.DeleteScheduledJobResponse)) {
    throw new Error('Expected argument of type scheduledjob.DeleteScheduledJobResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_scheduledjob_DeleteScheduledJobResponse(buffer_arg) {
  return scheduled_job_pb.DeleteScheduledJobResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_scheduledjob_GetScheduledJobRequest(arg) {
  if (!(arg instanceof scheduled_job_pb.GetScheduledJobRequest)) {
    throw new Error('Expected argument of type scheduledjob.GetScheduledJobRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_scheduledjob_GetScheduledJobRequest(buffer_arg) {
  return scheduled_job_pb.GetScheduledJobRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_scheduledjob_GetScheduledJobResponse(arg) {
  if (!(arg instanceof scheduled_job_pb.GetScheduledJobResponse)) {
    throw new Error('Expected argument of type scheduledjob.GetScheduledJobResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_scheduledjob_GetScheduledJobResponse(buffer_arg) {
  return scheduled_job_pb.GetScheduledJobResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_scheduledjob_ListScheduledJobsRequest(arg) {
  if (!(arg instanceof scheduled_job_pb.ListScheduledJobsRequest)) {
    throw new Error('Expected argument of type scheduledjob.ListScheduledJobsRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_scheduledjob_ListScheduledJobsRequest(buffer_arg) {
  return scheduled_job_pb.ListScheduledJobsRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_scheduledjob_ListScheduledJobsResponse(arg) {
  if (!(arg instanceof scheduled_job_pb.ListScheduledJobsResponse)) {
    throw new Error('Expected argument of type scheduledjob.ListScheduledJobsResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_scheduledjob_ListScheduledJobsResponse(buffer_arg) {
  return scheduled_job_pb.ListScheduledJobsResponse.deserializeBinary(new Uint8Array(buffer_arg));
}


var ScheduledJobServiceService = exports.ScheduledJobServiceService = {
  createOneOffScheduledJob: {
    path: '/scheduledjob.ScheduledJobService/CreateOneOffScheduledJob',
    requestStream: false,
    responseStream: false,
    requestType: scheduled_job_pb.CreateOneOffScheduledJobRequest,
    responseType: scheduled_job_pb.CreateScheduledJobResponse,
    requestSerialize: serialize_scheduledjob_CreateOneOffScheduledJobRequest,
    requestDeserialize: deserialize_scheduledjob_CreateOneOffScheduledJobRequest,
    responseSerialize: serialize_scheduledjob_CreateScheduledJobResponse,
    responseDeserialize: deserialize_scheduledjob_CreateScheduledJobResponse,
  },
  createRecurringScheduledJob: {
    path: '/scheduledjob.ScheduledJobService/CreateRecurringScheduledJob',
    requestStream: false,
    responseStream: false,
    requestType: scheduled_job_pb.CreateRecurringScheduledJobRequest,
    responseType: scheduled_job_pb.CreateScheduledJobResponse,
    requestSerialize: serialize_scheduledjob_CreateRecurringScheduledJobRequest,
    requestDeserialize: deserialize_scheduledjob_CreateRecurringScheduledJobRequest,
    responseSerialize: serialize_scheduledjob_CreateScheduledJobResponse,
    responseDeserialize: deserialize_scheduledjob_CreateScheduledJobResponse,
  },
  getScheduledJob: {
    path: '/scheduledjob.ScheduledJobService/GetScheduledJob',
    requestStream: false,
    responseStream: false,
    requestType: scheduled_job_pb.GetScheduledJobRequest,
    responseType: scheduled_job_pb.GetScheduledJobResponse,
    requestSerialize: serialize_scheduledjob_GetScheduledJobRequest,
    requestDeserialize: deserialize_scheduledjob_GetScheduledJobRequest,
    responseSerialize: serialize_scheduledjob_GetScheduledJobResponse,
    responseDeserialize: deserialize_scheduledjob_GetScheduledJobResponse,
  },
  listScheduledJobs: {
    path: '/scheduledjob.ScheduledJobService/ListScheduledJobs',
    requestStream: false,
    responseStream: false,
    requestType: scheduled_job_pb.ListScheduledJobsRequest,
    responseType: scheduled_job_pb.ListScheduledJobsResponse,
    requestSerialize: serialize_scheduledjob_ListScheduledJobsRequest,
    requestDeserialize: deserialize_scheduledjob_ListScheduledJobsRequest,
    responseSerialize: serialize_scheduledjob_ListScheduledJobsResponse,
    responseDeserialize: deserialize_scheduledjob_ListScheduledJobsResponse,
  },
  deleteScheduledJob: {
    path: '/scheduledjob.ScheduledJobService/DeleteScheduledJob',
    requestStream: false,
    responseStream: false,
    requestType: scheduled_job_pb.DeleteScheduledJobRequest,
    responseType: scheduled_job_pb.DeleteScheduledJobResponse,
    requestSerialize: serialize_scheduledjob_DeleteScheduledJobRequest,
    requestDeserialize: deserialize_scheduledjob_DeleteScheduledJobRequest,
    responseSerialize: serialize_scheduledjob_DeleteScheduledJobResponse,
    responseDeserialize: deserialize_scheduledjob_DeleteScheduledJobResponse,
  },
};

exports.ScheduledJobServiceClient = grpc.makeGenericClientConstructor(ScheduledJobServiceService, 'ScheduledJobService');
