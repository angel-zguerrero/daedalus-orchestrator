import { Component, OnInit, Input } from '@angular/core';
import { CommonModule } from '@angular/common';
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
import { ReactiveFormsModule, FormsModule, FormBuilder, FormGroup, Validators } from '@angular/forms';
import { IconDirective } from '@coreui/icons-angular';
import * as XLSX from 'xlsx';

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
    IconDirective
  ]
})
export class ExchangesComponent implements OnInit {
  @Input() tenantId: string = '';
  
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
    { value: 'headers', label: 'Headers' },
    { value: 'dead-letter', label: 'Dead Letter' }
  ];

  // VNamespace properties
  vnamespaces: any[] = [];
  vnamespaceSearchQuery = '';
  loadingVNamespaces = false;

  // VNamespace filter properties
  filterVNamespaces: any[] = [];
  filterVNamespaceQuery = '';
  loadingFilterVNamespaces = false;
  selectedVNamespaceFilter = '';

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
      vnamespace: ['', Validators.required]
    });
    this.exchangeFormUpdate = this.fb.group({
      name: ['', Validators.required]
    });
  }

  ngOnInit(): void {
    if (this.tenantId) {
      this.cursors.push('');
      this.loadExchanges();
      this.loadVNamespaces();
      this.loadFilterVNamespaces();
    }
  }

  loadExchanges(cursor: string = '', isPrevious: boolean = false): void {
    if (!isPrevious && cursor) {
      this.cursors.push(cursor);
    }
    
    this.exchangesService.getExchanges(this.tenantId, cursor, this.pageSize, this.searchQuery, this.selectedVNamespaceFilter).subscribe({
      next: (response) => {
        this.exchanges = response.result.Entities || [];
        this.cursor = response.result.Cursor;
      },
      error: (error) => {
        this.showAlert = true;
        this.errorMessage = error.error?.message || 'Failed to load exchanges';
      }
    });
  }

  searchExchanges(): void {
    this.cursors = [''];
    this.loadExchanges();
  }

  loadVNamespaces(query: string = ''): void {
    this.loadingVNamespaces = true;
    this.vNamespacesService.getVNamespaces(this.tenantId, '', 20, query).subscribe({
      next: (response) => {
        this.vnamespaces = response.data || [];
        this.loadingVNamespaces = false;
      },
      error: (error) => {
        console.error('Failed to load VNamespaces:', error);
        this.vnamespaces = [];
        this.loadingVNamespaces = false;
      }
    });
  }

  loadFilterVNamespaces(query: string = ''): void {
    this.loadingFilterVNamespaces = true;
    this.vNamespacesService.getVNamespaces(this.tenantId, '', 50, query).subscribe({
      next: (response) => {
        this.filterVNamespaces = response.data || [];
        this.loadingFilterVNamespaces = false;
      },
      error: (error) => {
        console.error('Failed to load Filter VNamespaces:', error);
        this.filterVNamespaces = [];
        this.loadingFilterVNamespaces = false;
      }
    });
  }

  searchVNamespaces(): void {
    this.loadVNamespaces(this.vnamespaceSearchQuery);
  }

  onVNamespaceInputChange(event: any): void {
    const value = event.target.value;
    this.vnamespaceSearchQuery = value;
    if (value.length >= 2) {
      this.searchVNamespaces();
    } else if (value.length === 0) {
      this.loadVNamespaces();
    }
  }

  searchFilterVNamespaces(): void {
    this.loadFilterVNamespaces(this.filterVNamespaceQuery);
  }

  onFilterVNamespaceInputChange(event: any): void {
    const value = event.target.value;
    this.filterVNamespaceQuery = value;
    if (value.length >= 2) {
      this.searchFilterVNamespaces();
    } else if (value.length === 0) {
      this.loadFilterVNamespaces();
    }
  }

  onVNamespaceFilterChange(event: any): void {
    this.selectedVNamespaceFilter = event.target.value;
    this.applyFilters();
  }

  applyFilters(): void {
    this.cursors = [''];
    this.loadExchanges();
  }

  clearVNamespaceFilter(): void {
    this.selectedVNamespaceFilter = '';
    this.filterVNamespaceQuery = '';
    this.applyFilters();
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
    this.vnamespaceSearchQuery = '';
    this.loadVNamespaces();
    this.showAlert = false;
  }

  openEditModal(exchange: any): void {
    this.selectedExchange = exchange;
    this.exchangeFormUpdate.reset();
    this.exchangeFormUpdate.patchValue({
      name: exchange.Name
    });
    this.editModalVisible = true;
    this.showAlert = false;
  }

  openDeleteModal(exchange: any): void {
    this.selectedExchange = exchange;
    this.deleteModalVisible = true;
  }

  openDetailsModal(exchange: any): void {
    console.log('Selected exchange from table:', exchange);
    // Use the exchange data directly from the table instead of making an API call
    this.selectedExchange = exchange;
    this.detailsModalVisible = true;
    this.showAlert = false; // Clear any previous alerts
  }

  createExchange(): void {
    if (this.exchangeForm.valid) {
      this.exchangesService.createExchange(this.tenantId, this.exchangeForm.value).subscribe({
        next: () => {
          this.createModalVisible = false;
          this.loadExchanges();
          this.showAlert = false;
        },
        error: (error) => {
          this.showAlert = true;
          this.errorMessage = error.error?.message || 'Failed to create exchange';
        }
      });
    }
  }

  updateExchange(): void {
    if (this.exchangeFormUpdate.valid) {
      const exchangeData = {
        name: this.exchangeFormUpdate.value.name,
        code: this.selectedExchange.Code, // Preserve existing code (frontend cannot edit)
        type: this.selectedExchange.Type, // Preserve original type
        vnamespace: this.selectedExchange.VNamespace, // Preserve original vnamespace
        id: this.selectedExchange.ID
      };
      this.exchangesService.createExchange(this.tenantId, exchangeData).subscribe({
        next: () => {
          this.editModalVisible = false;
          this.loadExchanges();
          this.showAlert = false;
        },
        error: (error) => {
          this.showAlert = true;
          this.errorMessage = error.error?.message || 'Failed to update exchange';
        }
      });
    }
  }

  deleteExchange(): void {
    this.exchangesService.deleteExchange(this.tenantId, this.selectedExchange.ID).subscribe({
      next: () => {
        this.deleteModalVisible = false;
        this.loadExchanges();
      },
      error: (error) => {
        this.showAlert = true;
        this.errorMessage = error.error?.message || 'Failed to delete exchange';
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
      const exchanges = XLSX.utils.sheet_to_json(worksheet, { header: ['Name', 'Type', 'VNamespace'] });

      // Remove header row
      exchanges.shift();

      if (exchanges.length === 0) {
        this.showAlert = true;
        this.errorMessage = 'The uploaded file is empty.';
        this.loading = false;
        return;
      }

      this.exchangesService.bulkCreateExchanges(this.tenantId, { exchanges }).subscribe({
        next: () => {
          this.bulkUploadModalVisible = false;
          this.loadExchanges();
          this.showAlert = false;
          this.loading = false;
        },
        error: (error) => {
          this.showAlert = true;
          this.errorMessage = error.error?.message || 'Failed to upload exchanges';
          this.loading = false;
        }
      });
    };
    fileReader.readAsArrayBuffer(this.file);
  }

  getExchangeTypeColor(type: string): string {
    const typeColors: { [key: string]: string } = {
      'direct': 'primary',
      'fanout': 'success',
      'topic': 'warning',
      'headers': 'info',
      'dead-letter': 'danger'
    };
    return typeColors[type] || 'secondary';
  }
}
