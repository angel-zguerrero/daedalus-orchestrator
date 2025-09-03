import { Component, OnInit, Input } from '@angular/core';
import { CommonModule, AsyncPipe } from '@angular/common';
import { QueuesService } from '../services/queues.service';
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
    AsyncPipe
  ]
})
export class QueuesComponent implements OnInit {
  @Input() tenantId: string = '';
  
  queues: any[] = [];
  cursor = '';
  cursors: string[] = [];
  pageSize = 20;
  searchQuery = '';

  public createModalVisible = false;
  public editModalVisible = false;
  public deleteModalVisible = false;
  public detailsModalVisible = false;
  public bulkUploadModalVisible = false;

  public showAlert = false;
  public errorMessage = '';
  public loading = false;

  queueForm: FormGroup;
  queueFormUpdate: FormGroup;
  selectedQueue: any;

  queueTypes = [
    { value: 'standard', label: 'Standard' },
    { value: 'delayed', label: 'Delayed' },
    { value: 'dead-letter', label: 'Dead Letter' }
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

  // Valid queue types
  private validQueueTypes = ['standard', 'delayed', 'dead-letter'];

  // Custom validator for queue type
  private queueTypeValidator = (control: any) => {
    if (!control.value) return null;
    return this.validQueueTypes.includes(control.value) ? null : { invalidQueueType: true };
  };

  constructor(
    private queuesService: QueuesService,
    private vNamespacesService: VNamespacesService,
    private fb: FormBuilder
  ) {
    this.queueForm = this.fb.group({
      name: ['', Validators.required],
      code: ['', Validators.required],
      type: ['standard', [Validators.required, this.queueTypeValidator]],
      vnamespace: this.vnamespaceCtrl,
      ttlQueue: [0, [Validators.min(0)]],
      allowDuplicated: [true],
      maxAttempts: [1, [Validators.required, Validators.min(1)]],
      priorityType: ['normal', Validators.required],
      maxPriority: [1, [Validators.required, Validators.min(1), Validators.max(100)]]
    });
    this.queueFormUpdate = this.fb.group({
      name: ['', Validators.required],
      ttlQueue: [0, [Validators.min(0)]],
      allowDuplicated: [true],
      maxAttempts: [1, [Validators.required, Validators.min(1)]],
      priorityType: ['normal', Validators.required],
      maxPriority: [1, [Validators.required, Validators.min(1), Validators.max(100)]]
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
    if (this.tenantId) {
      this.cursors.push('');
      this.loadQueues();
    }
  }

  private _filterVNamespaces(value: string): Observable<any[]> {
    this.loadingVNamespaces = true;
    return this.vNamespacesService.getVNamespaces(this.tenantId, '', 20, value).pipe(
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
    
    this.queuesService.getQueues(this.tenantId, cursor, this.pageSize, this.searchQuery, this.selectedVNamespaceFilter).subscribe({
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
      ttlQueue: 0,
      allowDuplicated: true,
      maxAttempts: 1,
      priorityType: 'normal',
      maxPriority: 1
    });
    this.priorityType = 'normal';
    this.maxPriority = 1;
    this.desiredPriorityThresholds = [0]; // Initialize with one element for maxPriority = 1
    this.showAlert = false;
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
      ttlQueue: queue.TTLQueue || 0,
      allowDuplicated: queue.AllowDuplicated !== undefined ? queue.AllowDuplicated : true,
      maxAttempts: queue.MaxAttempts || 1,
      priorityType: calculatedPriorityType,
      maxPriority: actualMaxPriority
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
    
    this.editModalVisible = true;
    this.showAlert = false;
  }

  openDeleteModal(queue: any): void {
    this.selectedQueue = queue;
    this.deleteModalVisible = true;
  }

  openDetailsModal(queue: any): void {
    console.log('Selected queue from table:', queue);
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

      const queueData = {
        ...formValue,
        desiredPriorityThresholds,
        maxPriority: this.maxPriority
      };

      this.queuesService.createQueue(this.tenantId, queueData).subscribe({
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

      const queueData = {
        name: formValue.name,
        code: this.selectedQueue.Code, // Preserve existing code (frontend cannot edit)
        type: this.selectedQueue.Type, // Preserve original type
        vnamespace: this.selectedQueue.VNamespace, // Preserve original vnamespace
        id: this.selectedQueue.ID,
        ttlQueue: formValue.ttlQueue,
        allowDuplicated: formValue.allowDuplicated,
        maxAttempts: formValue.maxAttempts,
        priorityType: formValue.priorityType,
        maxPriority: this.updateMaxPriority,
        desiredPriorityThresholds
      };
      
      this.queuesService.createQueue(this.tenantId, queueData).subscribe({
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
    this.queuesService.deleteQueue(this.tenantId, this.selectedQueue.code, this.selectedQueue.vnamespace).subscribe({
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
          header: ['Name', 'Code', 'Type', 'VNamespace', 'TTLQueue', 'AllowDuplicated', 'MaxAttempts', 'PriorityType', 'MaxPriority', 'PriorityThresholds'] 
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

          // Process TTLQueue - ensure it's a number >= 0
          let ttlQueue = 0; // default value
          if (queue.TTLQueue !== undefined && queue.TTLQueue !== null && queue.TTLQueue !== '') {
            ttlQueue = parseInt(queue.TTLQueue, 10);
            if (isNaN(ttlQueue) || ttlQueue < 0) {
              throw new Error(`Row ${index + 2}: Invalid TTLQueue value: ${queue.TTLQueue}. Must be a number >= 0`);
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
              const thresholdValues = thresholdStr.split(',').map(v => parseInt(v.trim(), 10));
              
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
            ttlQueue: ttlQueue,
            allowDuplicated: allowDuplicated,
            maxAttempts: maxAttempts,
            priorityType: priorityType,
            maxPriority: maxPriority,
            desiredPriorityThresholds: desiredPriorityThresholds
          };
        });

        this.queuesService.bulkCreateQueues(this.tenantId, { queues: processedQueues }).subscribe({
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
      'standard': 'primary',
      'delayed': 'warning',
      'dead-letter': 'danger'
    };
    return typeColors[type] || 'secondary';
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
}
