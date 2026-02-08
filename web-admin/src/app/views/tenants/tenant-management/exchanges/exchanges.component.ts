import { Component, OnInit, Input } from '@angular/core';
import { CommonModule, AsyncPipe } from '@angular/common';
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
  BadgeComponent
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

@Component({
  selector: 'app-exchanges',
  templateUrl: './exchanges.component.html',
  styleUrls: ['./exchanges.component.scss'],
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
    AsyncPipe
  ]
})
export class ExchangesComponent implements OnInit {
  @Input() tenantCode: string = '';
  
  exchanges: any[] = [];
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

  exchangeForm: FormGroup;
  exchangeFormUpdate: FormGroup;
  sendMessageForm: FormGroup;
  selectedExchange: any;

  exchangeTypes = [
    { value: 'direct', label: 'Direct' },
    { value: 'fanout', label: 'Fanout' },
    { value: 'topic', label: 'Topic' },
    { value: 'headers', label: 'Headers' }
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

    // Header management variables
  exchangeHeaderKey: string = '';
  exchangeHeaderValue: string = '';
  exchangeHeaders: { key: string, value: string }[] = [];
  exchangeUpdateHeaders: { key: string, value: string }[] = [];

  // Send Message properties
  messageParameters: { key: string, value: string }[] = [];
  messageHeaders: { key: string, value: string }[] = [];
  selectedFile: File | null = null;

  public file: File | null = null;

  constructor(
    private exchangesService: ExchangesService,
    private vNamespacesService: VNamespacesService,
    private fb: FormBuilder
  ) {
    this.exchangeForm = this.fb.group({
      name: ['', Validators.required],
      code: ['', Validators.required],
      type: ['direct', Validators.required],
      vnamespace: this.vnamespaceCtrl
    });
    this.exchangeFormUpdate = this.fb.group({
      name: ['', Validators.required]
    });

    this.sendMessageForm = this.fb.group({
      messageId: [''], // Optional field
      routingKey: [''], // Optional field for routing key/pattern
      handler: ['', Validators.required], // Required field
      priority: [0, [Validators.min(0)]], // Optional, but must be >= 0 if provided
      contentType: [''], // Optional
      content: [''] // Optional
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
      this.loadExchanges();
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

  loadExchanges(cursor: string = '', isPrevious: boolean = false): void {
    if (!isPrevious && cursor) {
      this.cursors.push(cursor);
    }
    
    this.exchangesService.getExchanges(this.tenantCode, cursor, this.pageSize, this.searchQuery, this.selectedVNamespaceFilter).subscribe({
      next: (response) => {
        this.exchanges = response.result.Entities || [];
        console.log('Loaded exchanges with headers:', this.exchanges.map(e => ({ id: e.ID, headers: e.Headers || e.headers })));
        this.cursor = response.result.Cursor;
      },
      error: (error) => {
        this.showAlert = true;
        this.errorMessage = ErrorUtil.formatErrorMessage(error);
      }
    });
  }

  searchExchanges(): void {
    this.cursors = [''];
    this.loadExchanges();
  }

  onVNamespaceFilterChange(value: string): void {
    this.selectedVNamespaceFilter = value;
    this.applyFilters();
  }

  applyFilters(): void {
    this.cursors = [''];
    this.loadExchanges();
  }

  nextPage(): void {
    if (this.cursor) {
      this.loadExchanges(this.cursor);
    }
  }

  previousPage(): void {
    if (this.cursors.length > 1) {
      this.cursors.pop();
      this.loadExchanges(this.cursors[this.cursors.length - 1], true);
    }
  }

  openCreateModal(): void {
    this.createModalVisible = true;
    this.exchangeForm.reset();
    this.exchangeForm.patchValue({ type: 'direct' });
    this.exchangeHeaders = [];
    this.exchangeHeaderKey = '';
    this.exchangeHeaderValue = '';
    this.showAlert = false;
  }

  openEditModal(exchange: any): void {
    this.selectedExchange = exchange;
    this.exchangeFormUpdate.reset();
    this.exchangeFormUpdate.patchValue({
      name: exchange.Name
    });
    
    // Load existing headers - try both Headers (backend) and headers (frontend)
    this.exchangeUpdateHeaders = [];
    const headersData = exchange.Headers || exchange.headers || {};
    if (headersData && typeof headersData === 'object') {
      Object.keys(headersData).forEach(key => {
        this.exchangeUpdateHeaders.push({
          key: key,
          value: headersData[key]
        });
      });
    }
    
    this.exchangeHeaderKey = '';
    this.exchangeHeaderValue = '';
    this.editModalVisible = true;
    this.showAlert = false;
  }

  openDeleteModal(exchange: any): void {
    this.selectedExchange = exchange;
    this.deleteModalVisible = true;
  }

  openDetailsModal(exchange: any): void {
    console.log('Selected exchange from table:', exchange);
    console.log('Headers in exchange:', exchange?.Headers || exchange?.headers);
    // Use the exchange data directly from the table instead of making an API call
    this.selectedExchange = exchange;
    this.detailsModalVisible = true;
    this.showAlert = false; // Clear any previous alerts
  }

  // Helper method to get headers for display in details modal
  getExchangeHeadersForDisplay(exchange: any): { key: string, value: string }[] {
    if (!exchange) return [];
    const headers = exchange.Headers || exchange.headers || {};
    return this.getHeadersArray(headers);
  }

  createExchange(): void {
    if (this.exchangeForm.valid) {
      // Convert headers array to object
      const headersObj = this.getHeadersAsMap();
      const exchangeData = {
        ...this.exchangeForm.value,
        headers: headersObj
      };
      
      this.exchangesService.createExchange(this.tenantCode, exchangeData).subscribe({
        next: () => {
          this.createModalVisible = false;
          this.loadExchanges();
          this.showAlert = false;
        },
        error: (error) => {
          this.showAlert = true;
          this.errorMessage = ErrorUtil.formatErrorMessage(error);
        }
      });
    } else {
      // Mark all fields as touched to show validation errors
      this.exchangeForm.markAllAsTouched();
    }
  }

  updateExchange(): void {
    if (this.exchangeFormUpdate.valid) {
      // Convert headers array to object
      const headersObj = this.getUpdateHeadersAsMap();
      const exchangeData = {
        name: this.exchangeFormUpdate.value.name,
        code: this.selectedExchange.Code, // Preserve existing code (frontend cannot edit)
        type: this.selectedExchange.Type, // Preserve original type
        vnamespace: this.selectedExchange.VNamespace, // Preserve original vnamespace
        id: this.selectedExchange.ID,
        headers: headersObj
      };
      this.exchangesService.createExchange(this.tenantCode, exchangeData).subscribe({
        next: () => {
          this.editModalVisible = false;
          this.loadExchanges();
          this.showAlert = false;
        },
        error: (error) => {
          this.showAlert = true;
          this.errorMessage = ErrorUtil.formatErrorMessage(error);
        }
      });
    } else {
      // Mark all fields as touched to show validation errors
      this.exchangeFormUpdate.markAllAsTouched();
    }
  }

  deleteExchange(): void {
    this.exchangesService.deleteExchange(this.tenantCode, this.selectedExchange.Code, this.selectedExchange.VNamespace).subscribe({
      next: () => {
        this.deleteModalVisible = false;
        this.loadExchanges();
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

  onFileChange(event: any): void {
    this.file = event.target.files[0];
  }

  uploadExchanges(): void {
    if (!this.file) {
      this.showAlert = true;
      this.errorMessage = 'Please select a file to upload.';
      return;
    }

    this.loading = true;
    const fileReader = new FileReader();
    fileReader.onload = (e: any) => {
      const data = new Uint8Array(e.target.result);
      const workbook = XLSX.read(data, { type: 'array' });
      const worksheet = workbook.Sheets[workbook.SheetNames[0]];
      const rawExchanges = XLSX.utils.sheet_to_json(worksheet, { header: ['Name', 'Code', 'Type', 'VNamespace'] });

      // Remove header row
      rawExchanges.shift();

      if (rawExchanges.length === 0) {
        this.showAlert = true;
        this.errorMessage = 'The uploaded file is empty.';
        this.loading = false;
        return;
      }

      // Transform the data to match backend expectations
      const exchanges = rawExchanges.map((exchange: any) => ({
        name: exchange.Name,
        code: exchange.Code,
        type: exchange.Type,
        vnamespace: exchange.VNamespace
      }));

      this.exchangesService.bulkCreateExchanges(this.tenantCode, { exchanges }).subscribe({
        next: () => {
          this.bulkUploadModalVisible = false;
          this.loadExchanges();
          this.showAlert = false;
          this.loading = false;
        },
        error: (error) => {
          this.showAlert = true;
          this.errorMessage = ErrorUtil.formatErrorMessage(error);
          this.loading = false;
        }
      });
    };
    fileReader.readAsArrayBuffer(this.file);
  }

  // Headers management methods
  addExchangeHeader(): void {
    if (this.exchangeHeaderKey.trim() && this.exchangeHeaderValue.trim()) {
      // Check if we're in create mode or edit mode
      const targetArray = this.editModalVisible ? this.exchangeUpdateHeaders : this.exchangeHeaders;
      const existingIndex = targetArray.findIndex(h => h.key === this.exchangeHeaderKey.trim());
      
      if (existingIndex >= 0) {
        // Update existing header
        targetArray[existingIndex].value = this.exchangeHeaderValue.trim();
      } else {
        // Add new header
        targetArray.push({
          key: this.exchangeHeaderKey.trim(),
          value: this.exchangeHeaderValue.trim()
        });
      }
      this.exchangeHeaderKey = '';
      this.exchangeHeaderValue = '';
    }
  }

  removeExchangeHeader(index: number): void {
    // Check if we're in create mode or edit mode
    if (this.editModalVisible) {
      this.exchangeUpdateHeaders.splice(index, 1);
    } else {
      this.exchangeHeaders.splice(index, 1);
    }
  }

  private getHeadersAsMap(): { [key: string]: string } {
    const headersMap: { [key: string]: string } = {};
    this.exchangeHeaders.forEach(header => {
      if (header.key && header.value) {
        headersMap[header.key] = header.value;
      }
    });
    return headersMap;
  }

  private getUpdateHeadersAsMap(): { [key: string]: string } {
    const headersMap: { [key: string]: string } = {};
    this.exchangeUpdateHeaders.forEach(header => {
      if (header.key && header.value) {
        headersMap[header.key] = header.value;
      }
    });
    return headersMap;
  }

  // Helper method to convert headers object to array for display
  getHeadersArray(headers: { [key: string]: string }): { key: string, value: string }[] {
    if (!headers) return [];
    return Object.keys(headers).map(key => ({ key, value: headers[key] }));
  }

  getExchangeTypeColor(type: string): string {
    const typeColors: { [key: string]: string } = {
      'direct': 'primary',
      'fanout': 'success',
      'topic': 'warning',
      'headers': 'info'
    };
    return typeColors[type] || 'secondary';
  }

  // Helper methods for routing key field
  shouldShowRoutingKeyField(): boolean {
    if (!this.selectedExchange) return false;
    const exchangeType = this.selectedExchange.Type?.toLowerCase();
    return exchangeType === 'direct' || exchangeType === 'topic';
  }

  isRoutingKeyRequired(): boolean {
    if (!this.selectedExchange) return false;
    const exchangeType = this.selectedExchange.Type?.toLowerCase();
    return exchangeType === 'direct'; // Direct exchanges typically require routing key, topic can be optional
  }

  getRoutingKeyPlaceholder(): string {
    if (!this.selectedExchange) return 'Enter routing key or pattern';
    
    switch (this.selectedExchange.Type?.toLowerCase()) {
      case 'direct':
        return 'Enter exact routing key (e.g., user.created)';
      case 'topic':
        return 'Enter topic pattern (e.g., user.* or notifications.#)';
      case 'fanout':
        return 'Not used for fanout exchanges';
      case 'headers':
        return 'Not used for headers exchanges (uses headers instead)';
      default:
        return 'Enter routing key or pattern';
    }
  }

  getRoutingKeyHelpText(): string {
    if (!this.selectedExchange) return 'Routing information for message delivery';
    
    switch (this.selectedExchange.Type?.toLowerCase()) {
      case 'direct':
        return 'Direct exchange: exact match with bound queue routing keys';
      case 'topic':
        return 'Topic exchange: * matches one word, # matches zero or more words';
      case 'fanout':
        return 'Fanout exchange: messages sent to all bound queues (routing key ignored)';
      case 'headers':
        return 'Headers exchange: routing based on message headers (routing key ignored)';
      default:
        return 'Routing information for message delivery';
    }
  }

  // Send Message Modal Methods
  openSendMessageModal(exchange: any): void {
    this.selectedExchange = exchange;
    this.sendMessageForm.reset({
      messageId: '',
      routingKey: '',
      handler: '',
      priority: 0,
      contentType: '',
      content: ''
    });
    
    // Update routing key validation based on exchange type
    this.updateRoutingKeyValidation();
    
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

  private updateRoutingKeyValidation(): void {
    const routingKeyControl = this.sendMessageForm.get('routingKey');
    if (routingKeyControl) {
      // Clear existing validators
      routingKeyControl.clearValidators();
      
      // Add required validator only for direct exchanges
      if (this.isRoutingKeyRequired()) {
        routingKeyControl.setValidators([Validators.required]);
      }
      
      // Update the validity
      routingKeyControl.updateValueAndValidity();
    }
  }

  // Parameter management methods
  addParameter(): void {
    this.messageParameters.push({ key: '', value: '' });
  }

  removeParameter(index: number): void {
    this.messageParameters.splice(index, 1);
  }

  // Header management methods
  addHeader(): void {
    this.messageHeaders.push({ key: '', value: '' });
  }

  removeHeader(index: number): void {
    this.messageHeaders.splice(index, 1);
  }

  // Content type change handler
  onContentTypeChange(): void {
    const contentType = this.sendMessageForm.get('contentType')?.value;
    const contentControl = this.sendMessageForm.get('content');
    
    // Clear all validators first
    contentControl?.clearValidators();
    contentControl?.setValue('');
    
    // Add JSON validator only for JSON content type
    if (contentType === 'application/json') {
      contentControl?.addValidators(this.jsonValidator);
    }
    
    contentControl?.updateValueAndValidity();
    
    // Clear file selection when changing from binary to other types
    if (contentType !== 'application/octet-stream') {
      this.selectedFile = null;
    }
  }

  // JSON validator
  jsonValidator(control: any) {
    const value = control.value;
    if (!value) return null;
    
    try {
      JSON.parse(value);
      return null;
    } catch (e) {
      return { invalidJson: true };
    }
  }

  // File selection handler
  onFileSelect(event: any): void {
    const file = event.target.files[0];
    if (file) {
      this.selectedFile = file;
    }
  }

  // Helper method to format file size
  formatFileSize(bytes: number): string {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  }

  // Get parameters as map
  private getParametersAsMap(): { [key: string]: string } {
    const parametersMap: { [key: string]: string } = {};
    this.messageParameters.forEach(param => {
      if (param.key && param.value) {
        parametersMap[param.key] = param.value;
      }
    });
    return parametersMap;
  }

  // Get headers as map
  private getMessageHeadersAsMap(): { [key: string]: string } {
    const headersMap: { [key: string]: string } = {};
    this.messageHeaders.forEach(header => {
      if (header.key && header.value) {
        headersMap[header.key] = header.value;
      }
    });
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
      exchangeCode: this.selectedExchange.Code,
      routingKeyOrPatternOrQueueCode: this.sendMessageForm.get('routingKey')?.value || '',
      vnamespace: this.selectedExchange.VNamespace,
      message: {
        messageId: this.sendMessageForm.get('messageId')?.value || '',
        handler: this.sendMessageForm.get('handler')?.value,
        priority: this.sendMessageForm.get('priority')?.value || 0,
        contentType: this.sendMessageForm.get('contentType')?.value || '',
        parameters: this.getParametersAsMap(),
        headers: this.getMessageHeadersAsMap(),
        content: content ? (typeof content === 'string' ? Array.from(new TextEncoder().encode(content)) : content) : []
      }
    };

    console.log('Sending message payload:', messageData);
    console.log('Selected exchange:', this.selectedExchange);

    this.exchangesService.publishMessage(this.tenantCode, messageData).subscribe({
      next: (response) => {
        this.loading = false;
        this.sendMessageModalVisible = false;
        this.sendMessageForm.reset();
        this.messageParameters = [];
        this.messageHeaders = [];
        this.selectedFile = null;
        
        // Prepare message results for display in modal
        const queueMessages = response.queueMessages || {};
        this.messageResults = Object.entries(queueMessages)
          .map(([queueCode, messageId]) => ({ queueCode, messageId: messageId as string }));
        
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
}
