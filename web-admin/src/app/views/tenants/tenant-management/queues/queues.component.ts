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
      maxAttempts: [1, [Validators.required, Validators.min(1)]]
    });
    this.queueFormUpdate = this.fb.group({
      name: ['', Validators.required]
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
      maxAttempts: 1
    });
    this.showAlert = false;
  }

  openEditModal(queue: any): void {
    this.selectedQueue = queue;
    this.queueFormUpdate.reset();
    this.queueFormUpdate.patchValue({
      name: queue.Name
    });
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
      this.queuesService.createQueue(this.tenantId, this.queueForm.value).subscribe({
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
      const queueData = {
        name: this.queueFormUpdate.value.name,
        code: this.selectedQueue.Code, // Preserve existing code (frontend cannot edit)
        type: this.selectedQueue.Type, // Preserve original type
        vnamespace: this.selectedQueue.VNamespace, // Preserve original vnamespace
        id: this.selectedQueue.ID
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
    this.queuesService.deleteQueue(this.tenantId, this.selectedQueue.ID).subscribe({
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
      const data = new Uint8Array(e.target.result);
      const workbook = XLSX.read(data, { type: 'array' });
      const worksheet = workbook.Sheets[workbook.SheetNames[0]];
      const queues = XLSX.utils.sheet_to_json(worksheet, { header: ['Name', 'Code', 'Type', 'VNamespace'] });

      // Remove header row
      queues.shift();

      if (queues.length === 0) {
        this.showAlert = true;
        this.errorMessage = 'The uploaded file is empty.';
        this.loading = false;
        return;
      }

      this.queuesService.bulkCreateQueues(this.tenantId, { queues }).subscribe({
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
}
