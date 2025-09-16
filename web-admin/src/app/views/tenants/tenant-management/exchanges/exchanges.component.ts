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

  public showAlert = false;
  public errorMessage = '';
  public loading = false;

  exchangeForm: FormGroup;
  exchangeFormUpdate: FormGroup;
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
      const exchanges = XLSX.utils.sheet_to_json(worksheet, { header: ['Name', 'Code', 'Type', 'VNamespace'] });

      // Remove header row
      exchanges.shift();

      if (exchanges.length === 0) {
        this.showAlert = true;
        this.errorMessage = 'The uploaded file is empty.';
        this.loading = false;
        return;
      }

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
}
