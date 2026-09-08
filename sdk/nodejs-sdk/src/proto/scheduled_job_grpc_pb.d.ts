// package: scheduledjob
// file: scheduled_job.proto

/* tslint:disable */
/* eslint-disable */

import * as grpc from "@grpc/grpc-js";
import * as scheduled_job_pb from "./scheduled_job_pb";

interface IScheduledJobServiceService extends grpc.ServiceDefinition<grpc.UntypedServiceImplementation> {
    createOneOffScheduledJob: IScheduledJobServiceService_ICreateOneOffScheduledJob;
    createRecurringScheduledJob: IScheduledJobServiceService_ICreateRecurringScheduledJob;
    getScheduledJob: IScheduledJobServiceService_IGetScheduledJob;
    listScheduledJobs: IScheduledJobServiceService_IListScheduledJobs;
    deleteScheduledJob: IScheduledJobServiceService_IDeleteScheduledJob;
}

interface IScheduledJobServiceService_ICreateOneOffScheduledJob extends grpc.MethodDefinition<scheduled_job_pb.CreateOneOffScheduledJobRequest, scheduled_job_pb.CreateScheduledJobResponse> {
    path: "/scheduledjob.ScheduledJobService/CreateOneOffScheduledJob";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<scheduled_job_pb.CreateOneOffScheduledJobRequest>;
    requestDeserialize: grpc.deserialize<scheduled_job_pb.CreateOneOffScheduledJobRequest>;
    responseSerialize: grpc.serialize<scheduled_job_pb.CreateScheduledJobResponse>;
    responseDeserialize: grpc.deserialize<scheduled_job_pb.CreateScheduledJobResponse>;
}
interface IScheduledJobServiceService_ICreateRecurringScheduledJob extends grpc.MethodDefinition<scheduled_job_pb.CreateRecurringScheduledJobRequest, scheduled_job_pb.CreateScheduledJobResponse> {
    path: "/scheduledjob.ScheduledJobService/CreateRecurringScheduledJob";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<scheduled_job_pb.CreateRecurringScheduledJobRequest>;
    requestDeserialize: grpc.deserialize<scheduled_job_pb.CreateRecurringScheduledJobRequest>;
    responseSerialize: grpc.serialize<scheduled_job_pb.CreateScheduledJobResponse>;
    responseDeserialize: grpc.deserialize<scheduled_job_pb.CreateScheduledJobResponse>;
}
interface IScheduledJobServiceService_IGetScheduledJob extends grpc.MethodDefinition<scheduled_job_pb.GetScheduledJobRequest, scheduled_job_pb.GetScheduledJobResponse> {
    path: "/scheduledjob.ScheduledJobService/GetScheduledJob";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<scheduled_job_pb.GetScheduledJobRequest>;
    requestDeserialize: grpc.deserialize<scheduled_job_pb.GetScheduledJobRequest>;
    responseSerialize: grpc.serialize<scheduled_job_pb.GetScheduledJobResponse>;
    responseDeserialize: grpc.deserialize<scheduled_job_pb.GetScheduledJobResponse>;
}
interface IScheduledJobServiceService_IListScheduledJobs extends grpc.MethodDefinition<scheduled_job_pb.ListScheduledJobsRequest, scheduled_job_pb.ListScheduledJobsResponse> {
    path: "/scheduledjob.ScheduledJobService/ListScheduledJobs";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<scheduled_job_pb.ListScheduledJobsRequest>;
    requestDeserialize: grpc.deserialize<scheduled_job_pb.ListScheduledJobsRequest>;
    responseSerialize: grpc.serialize<scheduled_job_pb.ListScheduledJobsResponse>;
    responseDeserialize: grpc.deserialize<scheduled_job_pb.ListScheduledJobsResponse>;
}
interface IScheduledJobServiceService_IDeleteScheduledJob extends grpc.MethodDefinition<scheduled_job_pb.DeleteScheduledJobRequest, scheduled_job_pb.DeleteScheduledJobResponse> {
    path: "/scheduledjob.ScheduledJobService/DeleteScheduledJob";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<scheduled_job_pb.DeleteScheduledJobRequest>;
    requestDeserialize: grpc.deserialize<scheduled_job_pb.DeleteScheduledJobRequest>;
    responseSerialize: grpc.serialize<scheduled_job_pb.DeleteScheduledJobResponse>;
    responseDeserialize: grpc.deserialize<scheduled_job_pb.DeleteScheduledJobResponse>;
}

export const ScheduledJobServiceService: IScheduledJobServiceService;

export interface IScheduledJobServiceServer extends grpc.UntypedServiceImplementation {
    createOneOffScheduledJob: grpc.handleUnaryCall<scheduled_job_pb.CreateOneOffScheduledJobRequest, scheduled_job_pb.CreateScheduledJobResponse>;
    createRecurringScheduledJob: grpc.handleUnaryCall<scheduled_job_pb.CreateRecurringScheduledJobRequest, scheduled_job_pb.CreateScheduledJobResponse>;
    getScheduledJob: grpc.handleUnaryCall<scheduled_job_pb.GetScheduledJobRequest, scheduled_job_pb.GetScheduledJobResponse>;
    listScheduledJobs: grpc.handleUnaryCall<scheduled_job_pb.ListScheduledJobsRequest, scheduled_job_pb.ListScheduledJobsResponse>;
    deleteScheduledJob: grpc.handleUnaryCall<scheduled_job_pb.DeleteScheduledJobRequest, scheduled_job_pb.DeleteScheduledJobResponse>;
}

export interface IScheduledJobServiceClient {
    createOneOffScheduledJob(request: scheduled_job_pb.CreateOneOffScheduledJobRequest, callback: (error: grpc.ServiceError | null, response: scheduled_job_pb.CreateScheduledJobResponse) => void): grpc.ClientUnaryCall;
    createOneOffScheduledJob(request: scheduled_job_pb.CreateOneOffScheduledJobRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: scheduled_job_pb.CreateScheduledJobResponse) => void): grpc.ClientUnaryCall;
    createOneOffScheduledJob(request: scheduled_job_pb.CreateOneOffScheduledJobRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: scheduled_job_pb.CreateScheduledJobResponse) => void): grpc.ClientUnaryCall;
    createRecurringScheduledJob(request: scheduled_job_pb.CreateRecurringScheduledJobRequest, callback: (error: grpc.ServiceError | null, response: scheduled_job_pb.CreateScheduledJobResponse) => void): grpc.ClientUnaryCall;
    createRecurringScheduledJob(request: scheduled_job_pb.CreateRecurringScheduledJobRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: scheduled_job_pb.CreateScheduledJobResponse) => void): grpc.ClientUnaryCall;
    createRecurringScheduledJob(request: scheduled_job_pb.CreateRecurringScheduledJobRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: scheduled_job_pb.CreateScheduledJobResponse) => void): grpc.ClientUnaryCall;
    getScheduledJob(request: scheduled_job_pb.GetScheduledJobRequest, callback: (error: grpc.ServiceError | null, response: scheduled_job_pb.GetScheduledJobResponse) => void): grpc.ClientUnaryCall;
    getScheduledJob(request: scheduled_job_pb.GetScheduledJobRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: scheduled_job_pb.GetScheduledJobResponse) => void): grpc.ClientUnaryCall;
    getScheduledJob(request: scheduled_job_pb.GetScheduledJobRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: scheduled_job_pb.GetScheduledJobResponse) => void): grpc.ClientUnaryCall;
    listScheduledJobs(request: scheduled_job_pb.ListScheduledJobsRequest, callback: (error: grpc.ServiceError | null, response: scheduled_job_pb.ListScheduledJobsResponse) => void): grpc.ClientUnaryCall;
    listScheduledJobs(request: scheduled_job_pb.ListScheduledJobsRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: scheduled_job_pb.ListScheduledJobsResponse) => void): grpc.ClientUnaryCall;
    listScheduledJobs(request: scheduled_job_pb.ListScheduledJobsRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: scheduled_job_pb.ListScheduledJobsResponse) => void): grpc.ClientUnaryCall;
    deleteScheduledJob(request: scheduled_job_pb.DeleteScheduledJobRequest, callback: (error: grpc.ServiceError | null, response: scheduled_job_pb.DeleteScheduledJobResponse) => void): grpc.ClientUnaryCall;
    deleteScheduledJob(request: scheduled_job_pb.DeleteScheduledJobRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: scheduled_job_pb.DeleteScheduledJobResponse) => void): grpc.ClientUnaryCall;
    deleteScheduledJob(request: scheduled_job_pb.DeleteScheduledJobRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: scheduled_job_pb.DeleteScheduledJobResponse) => void): grpc.ClientUnaryCall;
}

export class ScheduledJobServiceClient extends grpc.Client implements IScheduledJobServiceClient {
    constructor(address: string, credentials: grpc.ChannelCredentials, options?: Partial<grpc.ClientOptions>);
    public createOneOffScheduledJob(request: scheduled_job_pb.CreateOneOffScheduledJobRequest, callback: (error: grpc.ServiceError | null, response: scheduled_job_pb.CreateScheduledJobResponse) => void): grpc.ClientUnaryCall;
    public createOneOffScheduledJob(request: scheduled_job_pb.CreateOneOffScheduledJobRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: scheduled_job_pb.CreateScheduledJobResponse) => void): grpc.ClientUnaryCall;
    public createOneOffScheduledJob(request: scheduled_job_pb.CreateOneOffScheduledJobRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: scheduled_job_pb.CreateScheduledJobResponse) => void): grpc.ClientUnaryCall;
    public createRecurringScheduledJob(request: scheduled_job_pb.CreateRecurringScheduledJobRequest, callback: (error: grpc.ServiceError | null, response: scheduled_job_pb.CreateScheduledJobResponse) => void): grpc.ClientUnaryCall;
    public createRecurringScheduledJob(request: scheduled_job_pb.CreateRecurringScheduledJobRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: scheduled_job_pb.CreateScheduledJobResponse) => void): grpc.ClientUnaryCall;
    public createRecurringScheduledJob(request: scheduled_job_pb.CreateRecurringScheduledJobRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: scheduled_job_pb.CreateScheduledJobResponse) => void): grpc.ClientUnaryCall;
    public getScheduledJob(request: scheduled_job_pb.GetScheduledJobRequest, callback: (error: grpc.ServiceError | null, response: scheduled_job_pb.GetScheduledJobResponse) => void): grpc.ClientUnaryCall;
    public getScheduledJob(request: scheduled_job_pb.GetScheduledJobRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: scheduled_job_pb.GetScheduledJobResponse) => void): grpc.ClientUnaryCall;
    public getScheduledJob(request: scheduled_job_pb.GetScheduledJobRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: scheduled_job_pb.GetScheduledJobResponse) => void): grpc.ClientUnaryCall;
    public listScheduledJobs(request: scheduled_job_pb.ListScheduledJobsRequest, callback: (error: grpc.ServiceError | null, response: scheduled_job_pb.ListScheduledJobsResponse) => void): grpc.ClientUnaryCall;
    public listScheduledJobs(request: scheduled_job_pb.ListScheduledJobsRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: scheduled_job_pb.ListScheduledJobsResponse) => void): grpc.ClientUnaryCall;
    public listScheduledJobs(request: scheduled_job_pb.ListScheduledJobsRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: scheduled_job_pb.ListScheduledJobsResponse) => void): grpc.ClientUnaryCall;
    public deleteScheduledJob(request: scheduled_job_pb.DeleteScheduledJobRequest, callback: (error: grpc.ServiceError | null, response: scheduled_job_pb.DeleteScheduledJobResponse) => void): grpc.ClientUnaryCall;
    public deleteScheduledJob(request: scheduled_job_pb.DeleteScheduledJobRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: scheduled_job_pb.DeleteScheduledJobResponse) => void): grpc.ClientUnaryCall;
    public deleteScheduledJob(request: scheduled_job_pb.DeleteScheduledJobRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: scheduled_job_pb.DeleteScheduledJobResponse) => void): grpc.ClientUnaryCall;
}
