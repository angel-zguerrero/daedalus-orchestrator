import { Component, OnInit, Input } from '@angular/core';
import { CommonModule, AsyncPipe } from '@angular/common';
import { QueuesService } from '../services/queues.service';
import { ExchangesService } from '../services/exchanges.service';
import { VNamespacesService } from '../services/vnamespaces.service';
import {
  TableModule,
  UtilitiesModule,
  ButtonModule,
  ModalModule,
  CardModule,
  FormModule,
  GridModule,
  AlertComponent,
  SpinnerComponent,
  BadgeComponent,
  ProgressModule
} from '@coreui/angular';
import { ReactiveFormsModule, FormsModule, FormBuilder, FormGroup, Validators, FormControl } from '@angular/forms';
import { IconDirective } from '@coreui/icons-angular';
import * as XLSX from 'xlsx';
import { MatAutocompleteModule } from '@angular/material/autocomplete';
import { MatInputModule } from '@angular/material/input';
import { MatFormFieldModule } from '@angular/material/form-field';
import { Observable, of } from 'rxjs';
import { startWith, map, debounceTime, switchMap } from 'rxjs/operators';
import { ErrorUtil } from '../../../../shared/utils/error.util';

interface Queue {
  ID: string;
  Name: string;
  Code: string;
  Type: string;
  State: string;
  VNamespace: string;
  DefaultQueueMessageTTL: number;
  DefaultQueueMessageDelayTime: number;
  QueueExpires: number;
  ExpireAt?: string;
  AllowDuplicated: boolean;
  MaxAttempts: number;
  MaxQueueSize: number;
  MessagesCount?: number;
  DesiredPriorityThresholds: { [key: number]: number };
  PriorityThresholds: { [key: number]: number };
  Headers?: { [key: string]: string };
  DeadLetterExchangeId?: string;
  DeadLetterExchangeRoutingKeyOrPattern?: string;
  CreatedAt: string;
  UpdatedAt: string;
}

interface Exchange {
  ID: string;
  Name: string;
  Code: string;
  Type: string;
  VNamespace: string;
  CreatedAt: string;
  UpdatedAt: string;
}

@Component({
  selector: 'app-queues',
  templateUrl: './queues.component.html',
  styleUrls: ['./queues.component.scss'],
  standalone: true,
  imports: [
    AlertComponent,
    CommonModule,
    TableModule,
    UtilitiesModule,
    ButtonModule,
    ModalModule,
    CardModule,
    FormModule,
    GridModule,
    ReactiveFormsModule,
    FormsModule,
    SpinnerComponent,
    BadgeComponent,
    IconDirective,
    MatFormFieldModule,
    MatInputModule,
    MatAutocompleteModule,
    AsyncPipe,
    ProgressModule
  ]
})
export class QueuesComponent implements OnInit {
  @Input() tenantCode: string = '';

  queues: Queue[] = [];
  exchanges: Exchange[] = [];
  filteredExchanges: Exchange[] = [];
  validExchangeTypes = ['direct', 'topic', 'fanout'];
  cursor = '';
  cursors: string[] = [];
  pageSize = 20;
  searchQuery = '';

  public createModalVisible = false;
  public editModalVisible = false;
  public deleteModalVisible = false;
  public detailsModalVisible = false;
  public bulkUploadModalVisible = false;
  public sendMessageModalVisible = false;
  public messageResultModalVisible = false;

  public showAlert = false;
  public errorMessage = '';
  public successMessage = '';
  public loading = false;

  // Message result properties
  public messageResults: { queueCode: string, messageId: string }[] = [];
  public messageSentSuccessfully = false;

  queueForm: FormGroup;
  queueFormUpdate: FormGroup;
  sendMessageForm: FormGroup;
  selectedQueue: any;

  queueTypes = [
    { value: 'standard', label: 'Standard' }
  ];

  // VNamespace properties
  vnamespaces: any[] = [];
  vnamespaceCtrl = new FormControl('');
  filteredVNamespaces: Observable<any[]>;
  loadingVNamespaces = false;

  // VNamespace filter properties
  vnamespaceFilterCtrl = new FormControl('');
  filteredFilterVNamespaces: Observable<any[]>;
  loadingFilterVNamespaces = false;
  selectedVNamespaceFilter = '';

  public file: File | null = null;

  // Send Message properties
  messageParameters: { key: string, value: string }[] = [];
  messageHeaders: { key: string, value: string }[] = [];
  selectedFile: File | null = null;
  messageParameterKey: string = '';
  messageParameterValue: string = '';
  messageHeaderKey: string = '';
  messageHeaderValue: string = '';

  // Priority management properties
  priorityType: string = 'normal';
  maxPriority: number = 1;
  desiredPriorityThresholds: number[] = [];

  // Update priority management properties
  updatePriorityType: string = 'normal';
  updateMaxPriority: number = 1;
  editDesiredPriorityThresholds: number[] = [];

  // Priority levels management
  priorityLevels: number = 3;
  editPriorityLevels: number = 3;

  // Headers management properties
  queueHeaders: { key: string, value: string }[] = [];
  queueHeaderKey: string = '';
  queueHeaderValue: string = '';

  // Valid queue types
  private validQueueTypes = ['standard'];

  // Custom validator for queue type
  private queueTypeValidator = (control: any) => {
    if (!control.value) return null;
    return this.validQueueTypes.includes(control.value) ? null : { invalidQueueType: true };
  };

  constructor(
    private queuesService: QueuesService,
    private exchangesService: ExchangesService,
    private vNamespacesService: VNamespacesService,
    private fb: FormBuilder
  ) {
    this.queueForm = this.fb.group({
      name: ['', Validators.required],
      code: ['', Validators.required],
      type: ['standard', [Validators.required, this.queueTypeValidator]],
      vnamespace: this.vnamespaceCtrl,
      defaultQueueMessageTTL: [0, [Validators.min(0)]],
      defaultQueueMessageDelayTime: [0, [Validators.min(0)]],
      queueExpires: [0, [Validators.min(0)]],
      allowDuplicated: [true],
      maxAttempts: [1, [Validators.required, Validators.min(1)]],
      maxQueueSize: [0, [Validators.min(0)]],
      priorityType: ['normal', Validators.required],
      maxPriority: [1, [Validators.required, Validators.min(1), Validators.max(100)]],
      deadLetterExchangeId: [''],
      deadLetterExchangeRoutingKeyOrPattern: ['']
    });

    // Add dynamic validation for routing key based on exchange type
    this.queueForm.get('deadLetterExchangeId')?.valueChanges.subscribe(exchangeId => {
      const routingKeyControl = this.queueForm.get('deadLetterExchangeRoutingKeyOrPattern');
      if (this.isRoutingKeyRequired(exchangeId)) {
        routingKeyControl?.setValidators([Validators.required]);
      } else {
        routingKeyControl?.clearValidators();
      }
      routingKeyControl?.updateValueAndValidity();
    });
    this.queueFormUpdate = this.fb.group({
      name: ['', Validators.required],
      defaultQueueMessageTTL: [0, [Validators.min(0)]],
      defaultQueueMessageDelayTime: [0, [Validators.min(0)]],
      queueExpires: [0, [Validators.min(0)]],
      allowDuplicated: [true],
      maxAttempts: [1, [Validators.required, Validators.min(1)]],
      maxQueueSize: [0, [Validators.min(0)]],
      priorityType: ['normal', Validators.required],
      maxPriority: [1, [Validators.required, Validators.min(1), Validators.max(100)]],
      deadLetterExchangeId: [''],
      deadLetterExchangeRoutingKeyOrPattern: ['']
    });

    // Add dynamic validation for routing key based on exchange type in update form
    this.queueFormUpdate.get('deadLetterExchangeId')?.valueChanges.subscribe(exchangeId => {
      const routingKeyControl = this.queueFormUpdate.get('deadLetterExchangeRoutingKeyOrPattern');
      if (this.isRoutingKeyRequired(exchangeId)) {
        routingKeyControl?.setValidators([Validators.required]);
      } else {
        routingKeyControl?.clearValidators();
      }
      routingKeyControl?.updateValueAndValidity();
    });

    this.sendMessageForm = this.fb.group({
      messageId: [''],
      handler: [''],
      priority: [0, [Validators.min(0)]],
      contentType: [''],
      content: ['']
    });

    this.filteredVNamespaces = this.vnamespaceCtrl.valueChanges.pipe(
      startWith(''),
      debounceTime(300),
      switchMap(value => this._filterVNamespaces(value || ''))
    );

    this.filteredFilterVNamespaces = this.vnamespaceFilterCtrl.valueChanges.pipe(
      startWith(''),
      debounceTime(300),
      switchMap(value => this._filterVNamespaces(value || ''))
    );
  }

  ngOnInit(): void {
    if (this.tenantCode) {
      this.cursors.push('');
      this.loadQueues();
      // Load exchanges for Dead Letter configuration
      setTimeout(() => {
        this.loadValidExchanges();
      }, 100); // Small delay to ensure tenant context is ready
    }
  }

  private _filterVNamespaces(value: string): Observable<any[]> {
    this.loadingVNamespaces = true;
    return this.vNamespacesService.getVNamespaces(this.tenantCode, '', 20, value).pipe(
      map(response => {
        this.loadingVNamespaces = false;
        return response.data || [];
      })
    );
  }

  loadQueues(cursor: string = '', isPrevious: boolean = false): void {
    if (!isPrevious && cursor) {
      this.cursors.push(cursor);
    }

    this.queuesService.getQueues(this.tenantCode, cursor, this.pageSize, this.searchQuery, this.selectedVNamespaceFilter, true).subscribe({
      next: (response) => {
        this.queues = response.result.Entities || [];
        this.cursor = response.result.Cursor;
      },
      error: (error) => {
        this.showAlert = true;
        this.errorMessage = ErrorUtil.formatErrorMessage(error);
      }
    });
  }

  searchQueues(): void {
    this.cursors = [''];
    this.loadQueues();
  }

  onVNamespaceFilterChange(value: string): void {
    this.selectedVNamespaceFilter = value;
    this.applyFilters();
    // Also reload exchanges when VNamespace filter changes
    this.loadValidExchanges();
  }

  applyFilters(): void {
    this.cursors = [''];
    this.loadQueues();
  }

  nextPage(): void {
    if (this.cursor) {
      this.loadQueues(this.cursor);
    }
  }

  previousPage(): void {
    if (this.cursors.length > 1) {
      this.cursors.pop();
      this.loadQueues(this.cursors[this.cursors.length - 1], true);
    }
  }

  openCreateModal(): void {
    this.createModalVisible = true;
    this.queueForm.reset();
    this.queueForm.patchValue({
      type: 'standard',
      defaultQueueMessageTTL: 0,
      defaultQueueMessageDelayTime: 0,
      queueExpires: 0,
      allowDuplicated: true,
      maxAttempts: 1,
      maxQueueSize: 0,
      priorityType: 'normal',
      maxPriority: 1,
      deadLetterExchangeId: '',
      deadLetterExchangeRoutingKeyOrPattern: ''
    });
    this.priorityType = 'normal';
    this.maxPriority = 1;
    this.desiredPriorityThresholds = [0]; // Initialize with one element for maxPriority = 1
    this.queueHeaders = []; // Clear headers
    this.queueHeaderKey = '';
    this.queueHeaderValue = '';
    this.showAlert = false;

    // Reload exchanges to make sure they're available
    this.loadValidExchanges();
  }

  openEditModal(queue: any): void {
    this.selectedQueue = queue;
    this.queueFormUpdate.reset();

    // Calculate the correct priority type based on thresholds
    const calculatedPriorityType = this.getCalculatedPriorityType(queue);

    // Calculate the actual max priority from the queue data
    const actualMaxPriority = this.getCalculatedMaxPriorityLevels(queue);

    // Set current values
    this.queueFormUpdate.patchValue({
      name: queue.Name,
      defaultQueueMessageTTL: queue.DefaultQueueMessageTTL || 0,
      defaultQueueMessageDelayTime: queue.DefaultQueueMessageDelayTime || 0,
      queueExpires: queue.QueueExpires || 0,
      allowDuplicated: queue.AllowDuplicated !== undefined ? queue.AllowDuplicated : true,
      maxAttempts: queue.MaxAttempts || 1,
      maxQueueSize: queue.MaxQueueSize || 0,
      priorityType: calculatedPriorityType,
      maxPriority: actualMaxPriority,
      deadLetterExchangeId: queue.DeadLetterExchangeId || '',
      deadLetterExchangeRoutingKeyOrPattern: queue.DeadLetterExchangeRoutingKeyOrPattern || ''
    });

    // Set update priority management state
    this.updatePriorityType = calculatedPriorityType;
    this.updateMaxPriority = actualMaxPriority;

    // Set priority thresholds if they exist
    if (queue.DesiredPriorityThresholds && this.updatePriorityType === 'fair') {
      this.editDesiredPriorityThresholds = this.getPriorityThresholdsArray(queue.DesiredPriorityThresholds);
    } else if (this.updatePriorityType === 'normal') {
      // For normal priority, auto-generate the thresholds array based on maxPriority
      this.editDesiredPriorityThresholds = [];
      for (let i = 0; i < this.updateMaxPriority; i++) {
        this.editDesiredPriorityThresholds.push(0);
      }
    } else {
      this.editDesiredPriorityThresholds = [];
    }

    // Load existing headers
    this.queueHeaders = [];
    if (queue.Headers) {
      Object.keys(queue.Headers).forEach(key => {
        this.queueHeaders.push({
          key: key,
          value: queue.Headers[key]
        });
      });
    }
    this.queueHeaderKey = '';
    this.queueHeaderValue = '';

    this.editModalVisible = true;
    this.showAlert = false;

    // Reload exchanges to make sure they're available
    this.loadValidExchanges();
  }

  openDeleteModal(queue: any): void {
    this.selectedQueue = queue;
    this.deleteModalVisible = true;
  }

  openDetailsModal(queue: any): void {
    // Use the queue data directly from the table instead of making an API call
    this.selectedQueue = queue;
    this.detailsModalVisible = true;
    this.showAlert = false; // Clear any previous alerts
  }

  createQueue(): void {
    if (this.queueForm.valid) {
      const formValue = this.queueForm.value;

      // Build priority thresholds based on priority type
      let desiredPriorityThresholds: { [key: number]: number } = {};
      if (formValue.priorityType === 'normal') {
        // For normal priority, create object with 0-indexed keys: {0: 0, 1: 0, 2: 0, 3: 0}
        for (let i = 0; i < this.maxPriority; i++) {
          desiredPriorityThresholds[i] = 0;
        }
      } else if (formValue.priorityType === 'fair') {
        // For fair priority, create object with 0-indexed keys: {0: value, 1: value, etc}
        for (let i = 0; i < this.maxPriority; i++) {
          desiredPriorityThresholds[i] = this.desiredPriorityThresholds[i] || 0;
        }
      }

      // Convert headers array to object
      const headersObj: { [key: string]: string } = {};
      this.queueHeaders.forEach(header => {
        headersObj[header.key] = header.value;
      });

      const queueData = {
        ...formValue,
        desiredPriorityThresholds,
        maxPriority: this.maxPriority,
        headers: headersObj
      };

      this.queuesService.createQueue(this.tenantCode, queueData).subscribe({
        next: () => {
          this.createModalVisible = false;
          this.loadQueues();
          this.showAlert = false;
        },
        error: (error) => {
          this.showAlert = true;
          this.errorMessage = ErrorUtil.formatErrorMessage(error);
        }
      });
    } else {
      // Mark all fields as touched to show validation errors
      this.queueForm.markAllAsTouched();
    }
  }

  updateQueue(): void {
    if (this.queueFormUpdate.valid) {
      const formValue = this.queueFormUpdate.value;

      // Build priority thresholds based on priority type
      let desiredPriorityThresholds: { [key: number]: number } = {};
      if (formValue.priorityType === 'normal') {
        // For normal priority, create object with 0-indexed keys: {0: 0, 1: 0, 2: 0, 3: 0}
        for (let i = 0; i < this.updateMaxPriority; i++) {
          desiredPriorityThresholds[i] = 0;
        }
      } else if (formValue.priorityType === 'fair') {
        // For fair priority, create object with 0-indexed keys: {0: value, 1: value, etc}
        for (let i = 0; i < this.updateMaxPriority; i++) {
          desiredPriorityThresholds[i] = this.editDesiredPriorityThresholds[i] || 0;
        }
      }

      // Convert headers array to object
      const headersObj: { [key: string]: string } = {};
      this.queueHeaders.forEach(header => {
        headersObj[header.key] = header.value;
      });

      const queueData = {
        name: formValue.name,
        code: this.selectedQueue.Code, // Preserve existing code (frontend cannot edit)
        type: this.selectedQueue.Type, // Preserve original type
        vnamespace: this.selectedQueue.VNamespace, // Preserve original vnamespace
        id: this.selectedQueue.ID,
        defaultQueueMessageTTL: formValue.defaultQueueMessageTTL,
        defaultQueueMessageDelayTime: formValue.defaultQueueMessageDelayTime,
        queueExpires: formValue.queueExpires,
        allowDuplicated: formValue.allowDuplicated,
        maxAttempts: formValue.maxAttempts,
        maxQueueSize: formValue.maxQueueSize,
        priorityType: formValue.priorityType,
        maxPriority: this.updateMaxPriority,
        desiredPriorityThresholds,
        headers: headersObj,
        deadLetterExchangeId: formValue.deadLetterExchangeId,
        deadLetterExchangeRoutingKeyOrPattern: formValue.deadLetterExchangeRoutingKeyOrPattern
      };

      this.queuesService.createQueue(this.tenantCode, queueData).subscribe({
        next: () => {
          this.editModalVisible = false;
          this.loadQueues();
          this.showAlert = false;
        },
        error: (error) => {
          this.showAlert = true;
          this.errorMessage = ErrorUtil.formatErrorMessage(error);
        }
      });
    } else {
      // Mark all fields as touched to show validation errors
      this.queueFormUpdate.markAllAsTouched();
    }
  }

  deleteQueue(): void {
    this.queuesService.deleteQueue(this.tenantCode, this.selectedQueue.Code, this.selectedQueue.VNamespace).subscribe({
      next: () => {
        this.deleteModalVisible = false;
        this.loadQueues();
      },
      error: (error) => {
        this.showAlert = true;
        this.errorMessage = ErrorUtil.formatErrorMessage(error);
      }
    });
  }

  openBulkUploadModal(): void {
    this.bulkUploadModalVisible = true;
    this.showAlert = false;
  }

  // Headers management methods
  addQueueHeader(): void {
    if (this.queueHeaderKey.trim() && this.queueHeaderValue.trim()) {
      // Check if header key already exists
      const existingIndex = this.queueHeaders.findIndex(h => h.key === this.queueHeaderKey.trim());
      if (existingIndex >= 0) {
        // Update existing header
        this.queueHeaders[existingIndex].value = this.queueHeaderValue.trim();
      } else {
        // Add new header
        this.queueHeaders.push({
          key: this.queueHeaderKey.trim(),
          value: this.queueHeaderValue.trim()
        });
      }
      this.queueHeaderKey = '';
      this.queueHeaderValue = '';
    }
  }

  removeQueueHeader(index: number): void {
    this.queueHeaders.splice(index, 1);
  }

  // Helper method to convert headers object to array for display
  getHeadersArray(headers: { [key: string]: string }): { key: string, value: string }[] {
    if (!headers) return [];
    return Object.keys(headers).map(key => ({ key, value: headers[key] }));
  }

  onFileChange(event: any): void {
    this.file = event.target.files[0];
  }

  uploadQueues(): void {
    if (!this.file) {
      this.showAlert = true;
      this.errorMessage = 'Please select a file to upload.';
      return;
    }

    this.loading = true;
    const fileReader = new FileReader();
    fileReader.onload = (e: any) => {
      try {
        const data = new Uint8Array(e.target.result);
        const workbook = XLSX.read(data, { type: 'array' });
        const worksheet = workbook.Sheets[workbook.SheetNames[0]];
        const rawQueues = XLSX.utils.sheet_to_json(worksheet, {
          header: ['Name', 'Code', 'Type', 'VNamespace', 'DefaultQueueMessageTTL', 'DefaultQueueMessageDelayTime', 'QueueExpires', 'AllowDuplicated', 'MaxAttempts', 'MaxQueueSize', 'PriorityType', 'MaxPriority', 'PriorityThresholds']
        });

        // Remove header row
        rawQueues.shift();

        if (rawQueues.length === 0) {
          this.showAlert = true;
          this.errorMessage = 'The uploaded file is empty.';
          this.loading = false;
          return;
        }

        // Process and validate the data
        const processedQueues = rawQueues.map((queue: any, index: number) => {
          // Validate queue type
          if (!this.validQueueTypes.includes(queue.Type)) {
            throw new Error(`Row ${index + 2}: Invalid queue type: ${queue.Type}. Valid types are: ${this.validQueueTypes.join(', ')}`);
          }

          // Process AllowDuplicated field - handle various formats
          let allowDuplicated = true; // default value
          if (queue.AllowDuplicated !== undefined && queue.AllowDuplicated !== null) {
            const allowDupStr = String(queue.AllowDuplicated).toLowerCase().trim();
            if (allowDupStr === 'false' || allowDupStr === '0' || allowDupStr === 'no') {
              allowDuplicated = false;
            } else if (allowDupStr === 'true' || allowDupStr === '1' || allowDupStr === 'yes') {
              allowDuplicated = true;
            } else {
              throw new Error(`Row ${index + 2}: Invalid AllowDuplicated value: ${queue.AllowDuplicated}. Must be true/false, yes/no, or 1/0`);
            }
          }

          // Process DefaultQueueMessageTTL - ensure it's a number >= 0
          let defaultQueueMessageTTL = 0; // default value
          if (queue.DefaultQueueMessageTTL !== undefined && queue.DefaultQueueMessageTTL !== null && queue.DefaultQueueMessageTTL !== '') {
            defaultQueueMessageTTL = parseInt(queue.DefaultQueueMessageTTL, 10);
            if (isNaN(defaultQueueMessageTTL) || defaultQueueMessageTTL < 0) {
              throw new Error(`Row ${index + 2}: Invalid DefaultQueueMessageTTL value: ${queue.DefaultQueueMessageTTL}. Must be a number >= 0`);
            }
          }

          // Process DefaultQueueMessageDelayTime - ensure it's a number >= 0
          let defaultQueueMessageDelayTime = 0; // default value
          if (queue.DefaultQueueMessageDelayTime !== undefined && queue.DefaultQueueMessageDelayTime !== null && queue.DefaultQueueMessageDelayTime !== '') {
            defaultQueueMessageDelayTime = parseInt(queue.DefaultQueueMessageDelayTime, 10);
            if (isNaN(defaultQueueMessageDelayTime) || defaultQueueMessageDelayTime < 0) {
              throw new Error(`Row ${index + 2}: Invalid DefaultQueueMessageDelayTime value: ${queue.DefaultQueueMessageDelayTime}. Must be a number >= 0`);
            }
          }

          // Process QueueExpires - ensure it's a number >= 0
          let queueExpires = 0; // default value
          if (queue.QueueExpires !== undefined && queue.QueueExpires !== null && queue.QueueExpires !== '') {
            queueExpires = parseInt(queue.QueueExpires, 10);
            if (isNaN(queueExpires) || queueExpires < 0) {
              throw new Error(`Row ${index + 2}: Invalid QueueExpires value: ${queue.QueueExpires}. Must be a number >= 0`);
            }
          }

          // Process MaxAttempts - ensure it's a number > 0
          let maxAttempts = 1; // default value
          if (queue.MaxAttempts !== undefined && queue.MaxAttempts !== null && queue.MaxAttempts !== '') {
            maxAttempts = parseInt(queue.MaxAttempts, 10);
            if (isNaN(maxAttempts) || maxAttempts <= 0) {
              throw new Error(`Row ${index + 2}: Invalid MaxAttempts value: ${queue.MaxAttempts}. Must be a number > 0`);
            }
          }

          // Process MaxQueueSize - ensure it's a number >= 0
          let maxQueueSize = 0; // default value
          if (queue.MaxQueueSize !== undefined && queue.MaxQueueSize !== null && queue.MaxQueueSize !== '') {
            maxQueueSize = parseInt(queue.MaxQueueSize, 10);
            if (isNaN(maxQueueSize) || maxQueueSize < 0) {
              throw new Error(`Row ${index + 2}: Invalid MaxQueueSize value: ${queue.MaxQueueSize}. Must be a number >= 0`);
            }
          }

          // Process PriorityType
          let priorityType = 'normal'; // default value
          if (queue.PriorityType !== undefined && queue.PriorityType !== null && queue.PriorityType !== '') {
            const priorityTypeStr = String(queue.PriorityType).toLowerCase().trim();
            if (priorityTypeStr === 'normal' || priorityTypeStr === 'fair') {
              priorityType = priorityTypeStr;
            } else {
              throw new Error(`Row ${index + 2}: Invalid PriorityType value: ${queue.PriorityType}. Must be 'normal' or 'fair'`);
            }
          }

          // Process MaxPriority
          let maxPriority = 1; // default value
          if (queue.MaxPriority !== undefined && queue.MaxPriority !== null && queue.MaxPriority !== '') {
            maxPriority = parseInt(queue.MaxPriority, 10);
            if (isNaN(maxPriority) || maxPriority < 1 || maxPriority > 100) {
              throw new Error(`Row ${index + 2}: Invalid MaxPriority value: ${queue.MaxPriority}. Must be a number between 1 and 100`);
            }
          }

          // Process PriorityThresholds (legacy column name, but we use desiredPriorityThresholds)
          let desiredPriorityThresholds: { [key: number]: number } = {};

          if (priorityType === 'normal') {
            // For normal priority, create object with 0-indexed keys: {0: 0, 1: 0, 2: 0, 3: 0}
            for (let i = 0; i < maxPriority; i++) {
              desiredPriorityThresholds[i] = 0;
            }
          } else if (priorityType === 'fair' && queue.PriorityThresholds !== undefined && queue.PriorityThresholds !== null && queue.PriorityThresholds !== '') {
            try {
              const thresholdStr = String(queue.PriorityThresholds).trim();
              const delimiter = thresholdStr.includes('|') ? '|' : ',';
              const thresholdValues = thresholdStr.split(delimiter).map(v => parseInt(v.trim(), 10));

              if (thresholdValues.length !== maxPriority) {
                throw new Error(`Row ${index + 2}: PriorityThresholds count (${thresholdValues.length}) must match MaxPriority (${maxPriority})`);
              }

              thresholdValues.forEach((value, idx) => {
                if (isNaN(value) || value < 0) {
                  throw new Error(`Row ${index + 2}: Invalid threshold value at position ${idx + 1}: ${value}. Must be >= 0`);
                }
                desiredPriorityThresholds[idx + 1] = value;
              });
            } catch (error: any) {
              throw new Error(`Row ${index + 2}: Error parsing PriorityThresholds: ${error.message}`);
            }
          } else if (priorityType === 'fair') {
            // Default thresholds for fair priority
            for (let i = 1; i <= maxPriority; i++) {
              desiredPriorityThresholds[i] = 0;
            }
          }

          return {
            name: queue.Name,
            code: queue.Code,
            type: queue.Type,
            vnamespace: queue.VNamespace,
            defaultQueueMessageTTL: defaultQueueMessageTTL,
            defaultQueueMessageDelayTime: defaultQueueMessageDelayTime,
            queueExpires: queueExpires,
            allowDuplicated: allowDuplicated,
            maxAttempts: maxAttempts,
            maxQueueSize: maxQueueSize,
            priorityType: priorityType,
            maxPriority: maxPriority,
            desiredPriorityThresholds: desiredPriorityThresholds
          };
        });

        this.queuesService.bulkCreateQueues(this.tenantCode, { queues: processedQueues }).subscribe({
          next: () => {
            this.bulkUploadModalVisible = false;
            this.loadQueues();
            this.showAlert = false;
            this.loading = false;
          },
          error: (error) => {
            this.showAlert = true;
            this.errorMessage = ErrorUtil.formatErrorMessage(error);
            this.loading = false;
          }
        });
      } catch (error: any) {
        this.showAlert = true;
        this.errorMessage = error.message || 'Error processing the uploaded file.';
        this.loading = false;
      }
    };
    fileReader.readAsArrayBuffer(this.file);
  }

  getQueueTypeColor(type: string): string {
    const typeColors: { [key: string]: string } = {
      'standard': 'primary'
    };
    return typeColors[type] || 'secondary';
  }

  getQueueStateColor(state: string): string {
    const stateColors: { [key: string]: string } = {
      'QueueActive': 'success',
      'QueuePaused': 'warning',
      'QueueDraining': 'info',
      'QueueStopped': 'danger'
    };
    return stateColors[state] || 'secondary';
  }

  getQueueStateLabel(state: string): string {
    const stateLabels: { [key: string]: string } = {
      'QueueActive': 'Active',
      'QueuePaused': 'Paused',
      'QueueDraining': 'Draining',
      'QueueStopped': 'Stopped'
    };
    return stateLabels[state] || state;
  }

  // Priority management methods
  onPriorityTypeChange(): void {
    const formValue = this.queueForm.get('priorityType')?.value;
    this.priorityType = formValue || 'normal';

    // Ensure the thresholds array is properly sized for the current maxPriority
    this.desiredPriorityThresholds = [];
    for (let i = 0; i < this.maxPriority; i++) {
      this.desiredPriorityThresholds.push(0);
    }
  }

  onMaxPriorityChange(): void {
    const formValue = this.queueForm.get('maxPriority')?.value;
    this.maxPriority = Number(formValue) || 1;

    // Resize the thresholds array to match the new maxPriority
    const currentLength = this.desiredPriorityThresholds.length;
    const targetLength = this.maxPriority;

    if (targetLength > currentLength) {
      // Add new thresholds with default value of 0
      for (let i = currentLength; i < targetLength; i++) {
        this.desiredPriorityThresholds.push(0);
      }
    } else if (targetLength < currentLength) {
      // Remove excess thresholds
      this.desiredPriorityThresholds = this.desiredPriorityThresholds.slice(0, targetLength);
    }
  }

  updateFairThresholds(): void {
    const currentLength = this.desiredPriorityThresholds.length;
    const targetLength = this.maxPriority;

    if (targetLength > currentLength) {
      // Add new thresholds with default value of 0
      for (let i = currentLength; i < targetLength; i++) {
        this.desiredPriorityThresholds.push(0);
      }
    } else if (targetLength < currentLength) {
      // Remove excess thresholds
      this.desiredPriorityThresholds = this.desiredPriorityThresholds.slice(0, targetLength);
    }
  }

  getPriorityArray(): any[] {
    return Array.from({ length: this.maxPriority });
  }

  updateDesiredPriorityThresholds(index: number, value: any): void {
    const numValue = Number(value) || 0;
    this.desiredPriorityThresholds[index] = numValue;
  }

  getPriorityTypeColor(priorityType: string): string {
    const colors: { [key: string]: string } = {
      'normal': 'primary',
      'fair': 'success'
    };
    return colors[priorityType] || 'secondary';
  }

  getPriorityTypeLabel(priorityType: string): string {
    const labels: { [key: string]: string } = {
      'normal': 'Normal',
      'fair': 'Fair'
    };
    return labels[priorityType] || 'Unknown';
  }

  getPriorityThresholdsArray(thresholds: any): number[] {
    if (!thresholds) return [];

    if (Array.isArray(thresholds)) {
      return thresholds;
    }

    if (typeof thresholds === 'object') {
      // Convert object to array based on keys
      const result: number[] = [];
      Object.keys(thresholds).sort((a, b) => Number(a) - Number(b)).forEach(key => {
        result.push(thresholds[key]);
      });
      return result;
    }

    return [];
  }

  // Update priority management methods
  onUpdatePriorityTypeChange(): void {
    const formValue = this.queueFormUpdate.get('priorityType')?.value;
    this.updatePriorityType = formValue || 'normal';

    if (this.updatePriorityType === 'normal') {
      // For normal priority, auto-generate editDesiredPriorityThresholds based on updateMaxPriority
      // with all values set to 0 (0-indexed array)
      this.editDesiredPriorityThresholds = [];
      for (let i = 0; i < this.updateMaxPriority; i++) {
        this.editDesiredPriorityThresholds.push(0);
      }
    } else if (this.updatePriorityType === 'fair') {
      this.updateUpdateFairThresholds();
    }
  }

  onUpdateMaxPriorityChange(): void {
    const formValue = this.queueFormUpdate.get('maxPriority')?.value;
    this.updateMaxPriority = Number(formValue) || 1;

    if (this.updatePriorityType === 'normal') {
      // For normal priority, regenerate editDesiredPriorityThresholds array based on new updateMaxPriority
      this.editDesiredPriorityThresholds = [];
      for (let i = 0; i < this.updateMaxPriority; i++) {
        this.editDesiredPriorityThresholds.push(0);
      }
    } else if (this.updatePriorityType === 'fair') {
      this.updateUpdateFairThresholds();
    }
  }

  updateUpdateFairThresholds(): void {
    const currentLength = this.editDesiredPriorityThresholds.length;
    const targetLength = this.updateMaxPriority;

    if (targetLength > currentLength) {
      // Add new thresholds with default value of 0
      for (let i = currentLength; i < targetLength; i++) {
        this.editDesiredPriorityThresholds.push(0);
      }
    } else if (targetLength < currentLength) {
      // Remove excess thresholds
      this.editDesiredPriorityThresholds = this.editDesiredPriorityThresholds.slice(0, targetLength);
    }
  }

  getUpdatePriorityArray(): any[] {
    return Array.from({ length: this.updateMaxPriority });
  }

  updateEditDesiredPriorityThresholds(index: number, value: any): void {
    const numValue = Number(value) || 0;
    this.editDesiredPriorityThresholds[index] = numValue;
  }

  // Priority levels management functions
  setPriorityLevels(levels: number): void {
    this.priorityLevels = levels;
    this.desiredPriorityThresholds = Array(levels).fill(0);
  }

  getPriorityLevelsArray(): any[] {
    return Array.from({ length: this.priorityLevels });
  }

  setEditPriorityLevels(levels: number): void {
    this.editPriorityLevels = levels;
    this.editDesiredPriorityThresholds = Array(levels).fill(0);
  }

  getEditPriorityLevelsArray(): any[] {
    return Array.from({ length: this.editPriorityLevels });
  }

  getCalculatedPriorityType(queue: any): string {
    // Check DesiredPriorityThresholds first, then fallback to PriorityThresholds for backward compatibility
    const thresholds = queue.DesiredPriorityThresholds || queue.PriorityThresholds || queue.desiredPriorityThresholds || queue.priorityThresholds;

    if (thresholds && Object.keys(thresholds).length > 0) {
      // Check if all values in thresholds are 0
      const allValuesAreZero = Object.values(thresholds).every((value: any) => Number(value) === 0);

      if (allValuesAreZero) {
        return 'normal';
      } else {
        return 'fair';
      }
    }
    return 'normal';
  }

  getCalculatedMaxPriorityLevels(queue: any): number {
    // Check both DesiredPriorityThresholds and PriorityThresholds
    const desiredThresholds = queue.DesiredPriorityThresholds || queue.desiredPriorityThresholds;
    const currentThresholds = queue.PriorityThresholds || queue.priorityThresholds;

    // Use DesiredPriorityThresholds first, then fallback to PriorityThresholds
    const thresholds = desiredThresholds || currentThresholds;

    if (thresholds) {
      if (typeof thresholds === 'object' && !Array.isArray(thresholds)) {
        // Count the number of keys in the object
        return Object.keys(thresholds).length;
      } else if (Array.isArray(thresholds)) {
        return thresholds.length;
      }
    }

    // Fallback to MaxPriority or maxPriority if available
    return queue?.MaxPriority || queue?.maxPriority || 1;
  }

  // Send Message Modal Methods
  openSendMessageModal(queue: any): void {
    // Check if queue is expired before opening modal
    if (this.isQueueExpired(queue)) {
      alert('Cannot send messages to an expired queue. Please check the queue expiration status.');
      return;
    }

    this.selectedQueue = queue;
    this.sendMessageForm.reset({
      messageId: '',
      handler: '',
      priority: 0,
      contentType: '',
      content: ''
    });

    this.messageParameters = [];
    this.messageHeaders = [];
    this.selectedFile = null;
    this.sendMessageModalVisible = true;
    this.showAlert = false;
  }

  // Close message result modal
  closeMessageResultModal(): void {
    this.messageResultModalVisible = false;
    this.messageResults = [];
    this.messageSentSuccessfully = false;
  }

  // Parameter management methods
  addParameter(): void {
    if (this.messageParameterKey.trim() && this.messageParameterValue.trim()) {
      this.messageParameters.push({
        key: this.messageParameterKey.trim(),
        value: this.messageParameterValue.trim()
      });
      this.messageParameterKey = '';
      this.messageParameterValue = '';
    }
  }

  removeParameter(index: number): void {
    this.messageParameters.splice(index, 1);
  }

  // Header management methods
  addMessageHeader(): void {
    if (this.messageHeaderKey.trim() && this.messageHeaderValue.trim()) {
      this.messageHeaders.push({
        key: this.messageHeaderKey.trim(),
        value: this.messageHeaderValue.trim()
      });
      this.messageHeaderKey = '';
      this.messageHeaderValue = '';
    }
  }

  removeMessageHeader(index: number): void {
    this.messageHeaders.splice(index, 1);
  }

  // File upload methods
  onFileSelected(event: any): void {
    this.selectedFile = event.target.files[0];
    if (this.selectedFile) {
      // Automatically set content type for binary files
      this.sendMessageForm.patchValue({
        contentType: 'application/octet-stream'
      });
    }
  }

  // Helper methods for message sending
  private getParametersAsMap(): { [key: string]: string } {
    const parametersMap: { [key: string]: string } = {};
    for (const param of this.messageParameters) {
      parametersMap[param.key] = param.value;
    }
    return parametersMap;
  }

  private getMessageHeadersAsMap(): { [key: string]: string } {
    const headersMap: { [key: string]: string } = {};
    for (const header of this.messageHeaders) {
      headersMap[header.key] = header.value;
    }
    return headersMap;
  }

  // Send message method
  sendMessage(): void {
    if (this.sendMessageForm.invalid) {
      this.sendMessageForm.markAllAsTouched();
      return;
    }

    this.loading = true;
    this.showAlert = false;

    const contentType = this.sendMessageForm.get('contentType')?.value || '';
    let content: any = null;

    // Handle content based on type
    if (contentType === 'application/octet-stream' && this.selectedFile) {
      // For binary content, convert file to base64
      const reader = new FileReader();
      reader.onload = () => {
        const base64String = btoa(String.fromCharCode(...new Uint8Array(reader.result as ArrayBuffer)));
        this.sendMessageWithContent(base64String);
      };
      reader.readAsArrayBuffer(this.selectedFile);
      return;
    } else if (contentType && this.sendMessageForm.get('content')?.value) {
      content = this.sendMessageForm.get('content')?.value;
      if (contentType === 'application/json') {
        try {
          // Validate JSON and convert to string if it's an object
          content = typeof content === 'object' ? JSON.stringify(content) : content;
          JSON.parse(content); // Validate JSON format
        } catch (e) {
          this.errorMessage = 'Invalid JSON format in content';
          this.showAlert = true;
          this.loading = false;
          return;
        }
      }
    }

    this.sendMessageWithContent(content);
  }

  private sendMessageWithContent(content: any): void {
    // Prepare message data according to the API structure
    const messageData = {
      queueCode: this.selectedQueue.Code,
      vnamespace: this.selectedQueue.VNamespace,
      content: content || '',
      contentType: this.sendMessageForm.get('contentType')?.value || '',
      headers: this.getMessageHeadersAsMap(),
      priority: this.sendMessageForm.get('priority')?.value || 0,
      handler: this.sendMessageForm.get('handler')?.value || '',
      parameters: this.getParametersAsMap()
    };


    this.queuesService.enqueueMessage(this.tenantCode, messageData).subscribe({
      next: (response) => {
        this.loading = false;
        this.sendMessageModalVisible = false;
        this.sendMessageForm.reset();
        this.messageParameters = [];
        this.messageHeaders = [];
        this.selectedFile = null;

        // Prepare message results for display in modal
        this.messageResults = [{ queueCode: this.selectedQueue.Code, messageId: response.messageId }];

        this.messageSentSuccessfully = true;
        this.messageResultModalVisible = true;
      },
      error: (error) => {
        this.loading = false;
        console.error('Error sending message:', error);
        this.errorMessage = error.error?.error || 'Failed to send message. Please try again.';
        this.showAlert = true;
      }
    });
  }

  getExpirationInfo(queue: any): {
    show: boolean;
    isExpired: boolean;
    isNearExpiry: boolean;
    text: string;
    progressPercentage: number;
  } {
    if (!queue) {
      return {
        show: false,
        isExpired: false,
        isNearExpiry: false,
        text: '',
        progressPercentage: -1
      };
    }

    const now = new Date();

    // Check if queue has explicit ExpireAt
    if (queue.ExpireAt) {
      const expireAt = new Date(queue.ExpireAt);
      const isExpired = now > expireAt;

      if (isExpired) {
        return {
          show: true,
          isExpired: true,
          isNearExpiry: false,
          text: `Expired on ${expireAt.toLocaleString()}`,
          progressPercentage: 0
        };
      }

      const updatedAt = new Date(queue.UpdatedAt || queue.updatedAt);
      const totalTime = expireAt.getTime() - updatedAt.getTime();
      const remainingTime = expireAt.getTime() - now.getTime();
      const progressPercentage = totalTime > 0 ? (remainingTime / totalTime) * 100 : 0;
      const isNearExpiry = progressPercentage <= 20; // 20% or less remaining

      return {
        show: true,
        isExpired: false,
        isNearExpiry: isNearExpiry,
        text: `Expires on ${expireAt.toLocaleString()}`,
        progressPercentage: Math.max(0, progressPercentage)
      };
    }

    // Check if queue has QueueExpires setting
    const queueExpires = queue.QueueExpires || queue.queueExpires || 0;
    if (queueExpires > 0) {
      const updatedAt = new Date(queue.UpdatedAt || queue.updatedAt);
      const expireAt = new Date(updatedAt.getTime() + (queueExpires * 1000));
      const isExpired = now > expireAt;

      if (isExpired) {
        return {
          show: true,
          isExpired: true,
          isNearExpiry: false,
          text: `Expired on ${expireAt.toLocaleString()}`,
          progressPercentage: 0
        };
      }

      const totalTime = queueExpires * 1000; // Convert to milliseconds
      const elapsedTime = now.getTime() - updatedAt.getTime();
      const remainingTime = totalTime - elapsedTime;
      const progressPercentage = totalTime > 0 ? (remainingTime / totalTime) * 100 : 0;
      const isNearExpiry = progressPercentage <= 20; // 20% or less remaining

      return {
        show: true,
        isExpired: false,
        isNearExpiry: isNearExpiry,
        text: `Expires on ${expireAt.toLocaleString()}`,
        progressPercentage: Math.max(0, progressPercentage)
      };
    }

    // No expiration configured
    return {
      show: false,
      isExpired: false,
      isNearExpiry: false,
      text: '',
      progressPercentage: -1
    };
  }

  // Get selected exchange by ID
  getSelectedExchange(exchangeId: string): Exchange | null {
    const exchange = this.exchanges.find(exchange => exchange.ID === exchangeId) || null;
    return exchange;
  }

  // Check if routing key/pattern is required for selected exchange
  isRoutingKeyRequired(exchangeId: string): boolean {
    const exchange = this.getSelectedExchange(exchangeId);
    const required = exchange ? ['direct', 'topic'].includes(exchange.Type.toLowerCase()) : false;
    return required;
  }

  // Get exchange type label
  getExchangeTypeLabel(type: string): string {
    switch (type.toLowerCase()) {
      case 'direct': return 'Direct';
      case 'topic': return 'Topic';
      case 'fanout': return 'Fanout';
      default: return type;
    }
  }

  // Load valid exchanges for Dead Letter configuration
  loadValidExchanges() {

    // Use the same vnamespace filter as the queues, or empty string to get all
    const vnamespace = this.selectedVNamespaceFilter || '';

    this.exchangesService.getExchanges(this.tenantCode, '', 100, '', vnamespace).subscribe({
      next: (response) => {
        if (response && response.result && response.result.Entities) {
          // Filter only Direct, Topic, and Fanout exchanges
          this.exchanges = response.result.Entities.filter((exchange: any) =>
            this.validExchangeTypes.includes(exchange.Type.toLowerCase())
          );
          this.filteredExchanges = [...this.exchanges];
        } else {
          this.exchanges = [];
          this.filteredExchanges = [];
        }
      },
      error: (error) => {
        console.error('Error loading exchanges:', error);
        this.exchanges = [];
        this.filteredExchanges = [];
      }
    });
  }

  // Helper method to check if queue is expired (for other components)
  isQueueExpired(queue: any): boolean {
    return this.getExpirationInfo(queue).isExpired;
  }

  // Get color for messages count badge
  getMessagesCountColor(queue: any): string {
    const messagesCount = queue.MessagesCount || 0;
    const maxQueueSize = queue.MaxQueueSize || 0;

    if (messagesCount === 0) {
      return 'secondary';
    }

    if (maxQueueSize > 0) {
      const percentage = (messagesCount / maxQueueSize) * 100;
      if (percentage >= 80) {
        return 'danger';
      } else if (percentage >= 60) {
        return 'warning';
      } else if (percentage >= 30) {
        return 'info';
      } else {
        return 'success';
      }
    }

    // For unlimited queues, use different thresholds
    if (messagesCount >= 1000) {
      return 'warning';
    } else if (messagesCount >= 100) {
      return 'info';
    } else {
      return 'success';
    }
  }
}
