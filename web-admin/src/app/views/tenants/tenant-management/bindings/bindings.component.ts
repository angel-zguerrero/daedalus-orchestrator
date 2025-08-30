import { Component, OnInit, Input } from '@angular/core';
import { CommonModule, AsyncPipe } from '@angular/common';
import { BindingsService } from '../services/bindings.service';
import { ExchangesService } from '../services/exchanges.service';
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
import { MatAutocompleteModule } from '@angular/material/autocomplete';
import { MatInputModule } from '@angular/material/input';
import { MatFormFieldModule } from '@angular/material/form-field';
import { Observable, of } from 'rxjs';
import { startWith, map, debounceTime, switchMap } from 'rxjs/operators';
import { ErrorUtil } from '../../../../shared/utils/error.util';

@Component({
  selector: 'app-bindings',
  templateUrl: './bindings.component.html',
  styleUrls: ['./bindings.component.scss'],
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
export class BindingsComponent implements OnInit {
  @Input() tenantId: string = '';
  
  bindings: any[] = [];
  cursor = '';
  cursors: string[] = [];
  pageSize = 20;
  searchQuery = '';

  public createModalVisible = false;
  public deleteModalVisible = false;
  public detailsModalVisible = false;

  public showAlert = false;
  public errorMessage = '';
  public loading = false;

  bindingForm: FormGroup;
  selectedBinding: any;
  selectedExchange: any;
  selectedQueue: any;

  bindingTypes = [
    { value: 'classic', label: 'Classic' },
    { value: 'dynamic', label: 'Dynamic' }
  ];

  xMatchTypes = [
    { value: 'all', label: 'All' },
    { value: 'any', label: 'Any' }
  ];

  // VNamespace properties
  vnamespaces: any[] = [];
  vnamespaceCtrl = new FormControl('', Validators.required);
  filteredVNamespaces: Observable<any[]>;
  loadingVNamespaces = false;

  // VNamespace filter properties
  vnamespaceFilterCtrl = new FormControl('');
  filteredFilterVNamespaces: Observable<any[]>;
  loadingFilterVNamespaces = false;
  selectedVNamespaceFilter = '';

  // Exchange properties
  exchanges: any[] = [];
  exchangeCtrl = new FormControl('', Validators.required);
  filteredExchanges: Observable<any[]>;
  loadingExchanges = false;

  // Queue properties
  queues: any[] = [];
  queueCtrl = new FormControl('', Validators.required);
  filteredQueues: Observable<any[]>;
  loadingQueues = false;

  constructor(
    private bindingsService: BindingsService,
    private exchangesService: ExchangesService,
    private queuesService: QueuesService,
    private vNamespacesService: VNamespacesService,
    private fb: FormBuilder
  ) {
    this.bindingForm = this.fb.group({
      exchangeCode: this.exchangeCtrl,
      queueCode: this.queueCtrl,
      vnamespace: this.vnamespaceCtrl,
      routingKey: [''],
      pattern: [''],
      xMatch: ['all'],
      bindingType: ['classic', Validators.required]
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

    this.filteredExchanges = this.exchangeCtrl.valueChanges.pipe(
      startWith(''),
      debounceTime(300),
      switchMap(value => this._filterExchanges(value || ''))
    );

    this.filteredQueues = this.queueCtrl.valueChanges.pipe(
      startWith(''),
      debounceTime(300),
      switchMap(value => this._filterQueues(value || ''))
    );

    // Listen to VNamespace changes
    this.vnamespaceCtrl.valueChanges.subscribe(value => {
      this.onVNamespaceChange();
    });

    // Listen to exchange changes to update form validation
    this.exchangeCtrl.valueChanges.subscribe(value => {
      this.onExchangeChange(value || '');
    });

    // Listen to queue changes
    this.queueCtrl.valueChanges.subscribe(value => {
      this.onQueueChange(value || '');
    });
  }

  ngOnInit(): void {
    if (this.tenantId) {
      this.cursors.push('');
      this.loadBindings();
    }
  }

  private _filterVNamespaces(value: string): Observable<any[]> {
    this.loadingVNamespaces = true;
    return this.vNamespacesService.getVNamespaces(this.tenantId, '', 50, value).pipe(
      map(response => {
        this.loadingVNamespaces = false;
        return response.data || [];
      })
    );
  }

  private _filterExchanges(value: string): Observable<any[]> {
    this.loadingExchanges = true;
    const vnamespace = this.vnamespaceCtrl.value || '';
    return this.exchangesService.getExchanges(this.tenantId, '', 50, value, vnamespace).pipe(
      map(response => {
        this.loadingExchanges = false;
        return response.result?.Entities || [];
      })
    );
  }

  private _filterQueues(value: string): Observable<any[]> {
    this.loadingQueues = true;
    const vnamespace = this.vnamespaceCtrl.value || '';
    return this.queuesService.getQueues(this.tenantId, '', 50, value, vnamespace).pipe(
      map(response => {
        this.loadingQueues = false;
        return response.result?.Entities || [];
      })
    );
  }

  loadBindings(cursor: string = '', isPrevious: boolean = false): void {
    if (!isPrevious && cursor) {
      this.cursors.push(cursor);
    }
    
    this.bindingsService.getBindings(this.tenantId, cursor, this.pageSize, this.searchQuery, this.selectedVNamespaceFilter).subscribe({
      next: (response) => {
        this.bindings = response.result.Entities || [];
        this.cursor = response.result.Cursor;
      },
      error: (error) => {
        this.showAlert = true;
        this.errorMessage = ErrorUtil.formatErrorMessage(error);
      }
    });
  }

  searchBindings(): void {
    this.cursors = [''];
    this.loadBindings();
  }

  onVNamespaceFilterChange(value: string): void {
    this.selectedVNamespaceFilter = value;
    this.applyFilters();
  }

  applyFilters(): void {
    this.cursors = [''];
    this.loadBindings();
  }

  nextPage(): void {
    if (this.cursor) {
      this.loadBindings(this.cursor);
    }
  }

  previousPage(): void {
    if (this.cursors.length > 1) {
      this.cursors.pop();
      this.loadBindings(this.cursors[this.cursors.length - 1], true);
    }
  }

  openCreateModal(): void {
    this.createModalVisible = true;
    
    // Reset form and clear all values
    this.bindingForm.reset();
    
    // Clear the individual form controls
    this.vnamespaceCtrl.setValue('');
    this.exchangeCtrl.setValue('');
    this.queueCtrl.setValue('');
    
    // Set default values
    this.bindingForm.patchValue({ 
      bindingType: 'classic',
      xMatch: 'all'
    });
    
    // Reset selected entities
    this.selectedExchange = null;
    this.selectedQueue = null;
    
    this.showAlert = false;
  }

  openDeleteModal(binding: any): void {
    this.selectedBinding = binding;
    this.deleteModalVisible = true;
  }

  openDetailsModal(binding: any): void {
    console.log('Selected binding from table:', binding);
    this.selectedBinding = binding;
    this.detailsModalVisible = true;
    this.showAlert = false;
  }

  createBinding(): void {
    if (this.bindingForm.valid) {
      const bindingData = {
        exchangeCode: this.exchangeCtrl.value || '',
        queueCode: this.queueCtrl.value || '',
        vnamespace: this.vnamespaceCtrl.value || '',
        routingKey: this.bindingForm.get('routingKey')?.value || '',
        pattern: this.bindingForm.get('pattern')?.value || '',
        xMatch: this.bindingForm.get('xMatch')?.value || 'all',
        bindingType: this.bindingForm.get('bindingType')?.value || 'classic'
      };

      this.bindingsService.createBinding(this.tenantId, bindingData).subscribe({
        next: () => {
          this.createModalVisible = false;
          this.loadBindings();
          this.showAlert = false;
        },
        error: (error) => {
          this.showAlert = true;
          this.errorMessage = ErrorUtil.formatErrorMessage(error);
        }
      });
    } else {
      this.bindingForm.markAllAsTouched();
    }
  }

  deleteBinding(): void {
    if (this.selectedBinding) {
      this.bindingsService.deleteBinding(
        this.tenantId, 
        this.selectedBinding.ExchangeCode || this.selectedBinding.exchangeCode,
        this.selectedBinding.QueueCode || this.selectedBinding.queueCode,
        this.selectedBinding.VNamespace || this.selectedBinding.vnamespace
      ).subscribe({
        next: () => {
          this.deleteModalVisible = false;
          this.loadBindings();
        },
        error: (error) => {
          this.showAlert = true;
          this.errorMessage = ErrorUtil.formatErrorMessage(error);
        }
      });
    }
  }

  getBindingTypeColor(type: string): string {
    const typeColors: { [key: string]: string } = {
      'classic': 'primary',
      'dynamic': 'success'
    };
    return typeColors[type] || 'secondary';
  }

  getXMatchColor(xMatch: string): string {
    const xMatchColors: { [key: string]: string } = {
      'all': 'info',
      'any': 'warning'
    };
    return xMatchColors[xMatch] || 'secondary';
  }

  onVNamespaceChange(): void {
    // Reset exchange and queue when vnamespace changes
    this.exchangeCtrl.setValue('');
    this.queueCtrl.setValue('');
    this.selectedExchange = null;
    this.selectedQueue = null;
  }

  onExchangeChange(exchangeCode: string): void {
    if (exchangeCode) {
      // Find the selected exchange to get its type
      this.exchangesService.getExchanges(this.tenantId, '', 100, exchangeCode, this.vnamespaceCtrl.value || '').subscribe({
        next: (response) => {
          const exchanges = response.result?.Entities || [];
          this.selectedExchange = exchanges.find((ex: any) => ex.Code === exchangeCode);
          this.updateFormValidation();
        },
        error: (error) => {
          console.error('Error finding exchange:', error);
        }
      });
    } else {
      this.selectedExchange = null;
      this.updateFormValidation();
    }
  }

  onQueueChange(queueCode: string): void {
    if (queueCode) {
      // Find the selected queue
      this.queuesService.getQueues(this.tenantId, '', 100, queueCode, this.vnamespaceCtrl.value || '').subscribe({
        next: (response) => {
          const queues = response.result?.Entities || [];
          this.selectedQueue = queues.find((q: any) => q.Code === queueCode);
        },
        error: (error) => {
          console.error('Error finding queue:', error);
        }
      });
    } else {
      this.selectedQueue = null;
    }
  }

  updateFormValidation(): void {
    const routingKeyControl = this.bindingForm.get('routingKey');
    const patternControl = this.bindingForm.get('pattern');
    const xMatchControl = this.bindingForm.get('xMatch');

    // Reset validators
    routingKeyControl?.clearValidators();
    patternControl?.clearValidators();

    if (this.selectedExchange) {
      const exchangeType = this.selectedExchange.Type?.toLowerCase();
      
      switch (exchangeType) {
        case 'direct':
          routingKeyControl?.setValidators([Validators.required]);
          break;
        case 'topic':
          patternControl?.setValidators([Validators.required]);
          break;
        case 'headers':
          // xMatch is required for headers but we already have a default
          break;
        case 'fanout':
          // No additional validation needed for fanout
          break;
      }
    }

    routingKeyControl?.updateValueAndValidity();
    patternControl?.updateValueAndValidity();
  }

  isFieldRequired(fieldName: string): boolean {
    if (!this.selectedExchange) return false;
    
    const exchangeType = this.selectedExchange.Type?.toLowerCase();
    
    switch (fieldName) {
      case 'routingKey':
        return exchangeType === 'direct';
      case 'pattern':
        return exchangeType === 'topic';
      case 'xMatch':
        return exchangeType === 'headers';
      default:
        return false;
    }
  }

  isFieldVisible(fieldName: string): boolean {
    if (!this.selectedExchange) return true; // Show all fields when no exchange selected
    
    const exchangeType = this.selectedExchange.Type?.toLowerCase();
    
    switch (fieldName) {
      case 'routingKey':
        return exchangeType === 'direct';
      case 'pattern':
        return exchangeType === 'topic';
      case 'xMatch':
        return exchangeType === 'headers';
      default:
        return true;
    }
  }

  getExchangeTypeDisplayName(): string {
    if (!this.selectedExchange) return '';
    
    const type = this.selectedExchange.Type?.toLowerCase();
    switch (type) {
      case 'direct':
        return 'Direct (point-to-point routing)';
      case 'topic':
        return 'Topic (pattern-based routing)';
      case 'headers':
        return 'Headers (header-based routing)';
      case 'fanout':
        return 'Fanout (broadcast to all queues)';
      default:
        return this.selectedExchange.Type || 'Unknown';
    }
  }
}
