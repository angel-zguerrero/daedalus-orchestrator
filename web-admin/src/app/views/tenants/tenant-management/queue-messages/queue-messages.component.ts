import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router, RouterModule } from '@angular/router';
import {
  TableModule,
  UtilitiesModule,
  ButtonModule,
  CardModule,
  GridModule,
  AlertComponent,
  SpinnerComponent,
  BadgeComponent,
  ModalModule
} from '@coreui/angular';
import { IconDirective } from '@coreui/icons-angular';
import { QueueMessagesService } from '../services/queue-messages.service';
import { ErrorUtil } from '../../../../shared/utils/error.util';

interface QueueMessage {
  ID: string;
  MessageID: string;
  QueueID: string;
  QueuePartitionID: string;
  Priority: number;
  Attempts: number;
  ContentType: string;
  ContentLength: number;
  Content: string;
  Handler: string;
  Parameters: { [key: string]: string };
  VNamespace: string;
  CreatedAt: string;
  UpdatedAt: string;
}

@Component({
  selector: 'app-queue-messages',
  templateUrl: './queue-messages.component.html',
  styleUrls: ['./queue-messages.component.scss'],
  standalone: true,
  imports: [
    CommonModule,
    RouterModule,
    TableModule,
    UtilitiesModule,
    ButtonModule,
    CardModule,
    GridModule,
    AlertComponent,
    SpinnerComponent,
    BadgeComponent,
    ModalModule,
    IconDirective
  ]
})
export class QueueMessagesComponent implements OnInit {
  tenantCode: string = '';
  tenantName: string = '';
  queueCode: string = '';
  queueName: string = '';
  vnamespace: string = '';

  messages: QueueMessage[] = [];
  loading: boolean = false;
  showAlert: boolean = false;
  errorMessage: string = '';

  cursor: string = '';
  cursors: string[] = [''];
  pageSize: number = 20;

  selectedMessage: QueueMessage | null = null;
  detailsModalVisible: boolean = false;

  constructor(
    private route: ActivatedRoute,
    private router: Router,
    private queueMessagesService: QueueMessagesService
  ) {}

  ngOnInit(): void {
    this.route.params.subscribe(params => {
      this.tenantCode = params['tenantCode'];
      this.queueCode = params['queueCode'];
      this.vnamespace = params['vnamespace'];
      if (this.tenantCode && this.queueCode && this.vnamespace) {
        this.loadMessages();
      }
    });

    this.route.queryParams.subscribe(queryParams => {
      if (queryParams['queueName']) {
        this.queueName = queryParams['queueName'];
      }
      if (queryParams['tenantName']) {
        this.tenantName = queryParams['tenantName'];
      }
    });
  }

  loadMessages(): void {
    this.loading = true;
    this.showAlert = false;
    this.queueMessagesService
      .getQueueMessages(this.tenantCode, this.queueCode, this.vnamespace, this.cursor, this.pageSize)
      .subscribe({
        next: (response) => {
          this.loading = false;
          this.messages = response.result?.Entities || [];
          this.cursor = response.result?.Cursor || '';
        },
        error: (error) => {
          this.loading = false;
          this.showAlert = true;
          this.errorMessage = ErrorUtil.formatErrorMessage(error);
        }
      });
  }

  nextPage(): void {
    if (this.cursor) {
      this.cursors.push(this.cursor);
      this.loadMessages();
    }
  }

  previousPage(): void {
    if (this.cursors.length > 1) {
      this.cursors.pop();
      this.cursor = this.cursors[this.cursors.length - 1];
      this.loadMessages();
    }
  }

  openDetailsModal(message: QueueMessage): void {
    this.selectedMessage = message;
    this.detailsModalVisible = true;
  }

  goBackToQueues(): void {
    this.router.navigate(['/tenants', this.tenantCode, 'management'], {
      queryParams: {
        tab: 'queues',
        ...(this.tenantName ? { name: this.tenantName } : {})
      }
    });
  }

  getParametersDisplay(params: { [key: string]: string } | null): string {
    if (!params || Object.keys(params).length === 0) {
      return '-';
    }
    return Object.entries(params)
      .map(([k, v]) => `${k}: ${v}`)
      .join(', ');
  }

  getObjectKeys(obj: { [key: string]: string } | null): string[] {
    if (!obj) return [];
    return Object.keys(obj);
  }

  truncateContent(content: string, maxLength: number = 80): string {
    if (!content) return '-';
    return content.length > maxLength ? content.substring(0, maxLength) + '…' : content;
  }
}
