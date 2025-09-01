import { Component, OnInit, Input, ChangeDetectorRef } from '@angular/core';
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
  AlertModule, 
  BadgeModule
} from '@coreui/angular';
import { ReactiveFormsModule, FormsModule, FormBuilder, FormGroup, Validators, FormControl } from '@angular/forms';
import { IconDirective } from '@coreui/icons-angular';
import { MatAutocompleteModule, MatAutocompleteSelectedEvent } from '@angular/material/autocomplete';
import { MatInputModule } from '@angular/material/input';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatSelectModule } from '@angular/material/select';
import { Observable, of, combineLatest } from 'rxjs';
import { startWith, map, debounceTime, switchMap, catchError } from 'rxjs/operators';
import { ErrorUtil } from '../../../../shared/utils/error.util';

// Interfaces for models
interface VNamespace {
  Code: string;
  Name: string;
  Description?: string;
}

interface Exchange {
  Code: string;
  Name: string;
  Type: string;
  VNamespace: string;
  Description?: string;
}

interface Queue {
  Code: string;
  Name: string;
  VNamespace: string;
  Description?: string;
  Type?: string; // Added for template compatibility
}

interface Binding {
  ExchangeCode: string;
  QueueCode: string;
  VNamespace: string;
  RoutingKey?: string;
  Pattern?: string;
  XMatch?: string;
  BindingType: string;
  ID?: string;
  CreatedAt?: string;
  UpdatedAt?: string;
  // Compatibilidad con propiedades en camelCase
  exchangeCode?: string;
  queueCode?: string;
  vnamespace?: string;
  routingKey?: string;
  pattern?: string;
  xMatch?: string;
  bindingType?: string;
  id?: string;
  createdAt?: string;
  updatedAt?: string;
}

@Component({
  selector: 'app-bindings',
  templateUrl: './bindings.component.html',
  styleUrls: ['./bindings.component.scss'],
  standalone: true,
  imports: [
    CommonModule,
    AsyncPipe,
    TableModule,
    UtilitiesModule,
    ButtonModule,
    ModalModule,
    CardModule,
    FormModule,
    GridModule,
    AlertModule,
    BadgeModule,
    ReactiveFormsModule,
    FormsModule,
    IconDirective,
    MatAutocompleteModule,
    MatInputModule,
    MatFormFieldModule,
    MatSelectModule
  ],
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

  // Form and selected models
  bindingForm: FormGroup;
  selectedBinding: Binding | null = null;
  selectedVNamespace: VNamespace | null = null;
  selectedExchange: Exchange | null = null;
  selectedQueue: Queue | null = null;

  // Filter model for list
  selectedVNamespaceFilter: VNamespace | null = null;

  bindingTypes = [
    { value: 'classic', label: 'Classic' },
    { value: 'dynamic', label: 'Dynamic' }
  ];

  xMatchTypes = [
    { value: 'all', label: 'All' },
    { value: 'any', label: 'Any' }
  ];

  // Form Controls
  vnamespaceCtrl = new FormControl<VNamespace | null>(null, Validators.required);
  exchangeCtrl = new FormControl<Exchange | null>({ value: null, disabled: true }, Validators.required);
  queueCtrl = new FormControl<Queue | null>({ value: null, disabled: true }, Validators.required);
  vnamespaceFilterCtrl = new FormControl<VNamespace | null>(null);

  // Observables for autocompletes
  filteredVNamespaces!: Observable<VNamespace[]>;
  filteredExchanges!: Observable<Exchange[]>;
  filteredQueues!: Observable<Queue[]>;
  filteredFilterVNamespaces!: Observable<VNamespace[]>;

  // Loading states
  loadingVNamespaces = false;
  loadingExchanges = false;
  loadingQueues = false;

  constructor(
    private bindingsService: BindingsService,
    private exchangesService: ExchangesService,
    private queuesService: QueuesService,
    private vNamespacesService: VNamespacesService,
    private fb: FormBuilder,
    private cdr: ChangeDetectorRef
  ) {
    this.bindingForm = this.fb.group({
      vnamespace: this.vnamespaceCtrl,
      exchange: this.exchangeCtrl,
      queue: this.queueCtrl,
      routingKey: [''],
      pattern: [''],
      xMatch: ['all'],
      bindingType: ['classic', Validators.required]
    });

    this.setupAutocompletes();
    this.setupModelWatchers();
  }

  ngOnInit(): void {
    if (this.tenantId) {
      this.cursors.push('');
      this.loadBindings();
    }
  }

  private setupAutocompletes(): void {
    // VNamespace autocompletes
    this.filteredVNamespaces = this.vnamespaceCtrl.valueChanges.pipe(
      startWith(null),
      debounceTime(300),
      switchMap(value => this._filterVNamespaces(this.getSearchTerm(value)))
    );

    this.filteredFilterVNamespaces = this.vnamespaceFilterCtrl.valueChanges.pipe(
      startWith(null),
      debounceTime(300),
      switchMap(value => this._filterVNamespaces(this.getSearchTerm(value)))
    );

    // Exchange autocomplete
    this.filteredExchanges = this.exchangeCtrl.valueChanges.pipe(
      startWith(null),
      debounceTime(300),
      switchMap(value => this._filterExchanges(this.getSearchTerm(value)))
    );

    // Queue autocomplete
    this.filteredQueues = this.queueCtrl.valueChanges.pipe(
      startWith(null),
      debounceTime(300),
      switchMap(value => this._filterQueues(this.getSearchTerm(value)))
    );
  }

  private setupModelWatchers(): void {
    // Watch VNamespace changes
    this.vnamespaceCtrl.valueChanges.subscribe(vnamespace => {
      console.log('VNamespace FormControl value changed:', vnamespace);
      this.selectedVNamespace = vnamespace;
      this.onVNamespaceChange();
    });

    // Watch Exchange changes
    this.exchangeCtrl.valueChanges.subscribe(exchange => {
      console.log('Exchange FormControl value changed:', exchange);
      this.selectedExchange = exchange;
      this.onExchangeChange();
    });

    // Watch Queue changes
    this.queueCtrl.valueChanges.subscribe(queue => {
      console.log('Queue FormControl value changed:', queue);
      this.selectedQueue = queue;
      this.onQueueChange();
    });

    // Watch VNamespace filter changes
    this.vnamespaceFilterCtrl.valueChanges.subscribe(vnamespace => {
      this.selectedVNamespaceFilter = vnamespace;
      this.onVNamespaceFilterChange();
    });
  }

  private getSearchTerm(value: any): string {
    if (!value) return '';
    if (typeof value === 'string') return value;
    if (value && value.Code) return value.Code;
    if (value && value.Name) return value.Name;
    return '';
  }

  private _filterVNamespaces(value: string): Observable<VNamespace[]> {
    this.loadingVNamespaces = true;
    return this.vNamespacesService.getVNamespaces(this.tenantId, '', 50, value).pipe(
      map(response => {
        this.loadingVNamespaces = false;
        console.log('Raw vnamespaces response:', response);
        console.log('Response data:', response.data);
        
        return (response.data || []).map((item: any) => {
          console.log('Mapping vnamespace item:', item);
          
          // Intentar diferentes propiedades que podrían contener el código
          const mappedItem = {
            Code: item.Code || item.code || item.VirtualNamespaceCode || item.virtualNamespaceCode || item.Name || item.name,
            Name: item.Name || item.name || item.DisplayName || item.displayName || item.Code || item.code,
            Description: item.Description || item.description
          } as VNamespace;
          
          console.log('Mapped vnamespace item:', mappedItem);
          return mappedItem;
        });
      }),
      catchError(error => {
        this.loadingVNamespaces = false;
        console.error('Error filtering vnamespaces:', error);
        return of([]);
      })
    );
  }

  private _filterExchanges(value: string): Observable<Exchange[]> {
    if (!this.selectedVNamespace) {
      return of([]);
    }

    this.loadingExchanges = true;
    return this.exchangesService.getExchanges(this.tenantId, '', 50, value, this.selectedVNamespace.Code).pipe(
      map(response => {
        this.loadingExchanges = false;
        return (response.result?.Entities || []).map((item: any) => ({
          Code: item.Code,
          Name: item.Name,
          Type: item.Type,
          VNamespace: item.VNamespace,
          Description: item.Description
        } as Exchange));
      }),
      catchError(error => {
        this.loadingExchanges = false;
        console.error('Error filtering exchanges:', error);
        return of([]);
      })
    );
  }

  private _filterQueues(value: string): Observable<Queue[]> {
    if (!this.selectedVNamespace) {
      return of([]);
    }

    this.loadingQueues = true;
    return this.queuesService.getQueues(this.tenantId, '', 50, value, this.selectedVNamespace.Code).pipe(
      map(response => {
        this.loadingQueues = false;
        return (response.result?.Entities || []).map((item: any) => ({
          Code: item.Code,
          Name: item.Name,
          VNamespace: item.VNamespace,
          Description: item.Description
        } as Queue));
      }),
      catchError(error => {
        this.loadingQueues = false;
        console.error('Error filtering queues:', error);
        return of([]);
      })
    );
  }

  // Model change handlers
  private onVNamespaceChange(): void {
    console.log('VNamespace changed:', this.selectedVNamespace);
    
    // Reset dependent selections
    this.selectedExchange = null;
    this.selectedQueue = null;
    this.exchangeCtrl.setValue(null);
    this.queueCtrl.setValue(null);
    
    // Enable/disable controls based on vnamespace selection
    if (this.selectedVNamespace) {
      this.exchangeCtrl.enable();
      this.queueCtrl.enable();
    } else {
      this.exchangeCtrl.disable();
      this.queueCtrl.disable();
    }
    
    this.updateFormValidation();
    this.cdr.detectChanges();
  }

  private onExchangeChange(): void {
    console.log('Exchange changed:', this.selectedExchange);
    this.updateFormValidation();
    this.cdr.detectChanges();
  }

  private onQueueChange(): void {
    console.log('Queue changed:', this.selectedQueue);
  }

  // Display functions for autocompletes
  displayVNamespace = (vnamespace: VNamespace): string => {
    return vnamespace ? `${vnamespace.Code} - ${vnamespace.Name}` : '';
  }

  displayExchange = (exchange: Exchange): string => {
    return exchange ? `${exchange.Code} - ${exchange.Name}` : '';
  }

  displayQueue = (queue: Queue): string => {
    return queue ? `${queue.Code} - ${queue.Name}` : '';
  }

  // Validation and visibility methods
  get canSelectExchange(): boolean {
    return this.selectedVNamespace !== null;
  }

  get canSelectQueue(): boolean {
    return this.selectedVNamespace !== null;
  }

  get showRoutingKey(): boolean {
    return this.selectedExchange?.Type?.toLowerCase() === 'direct';
  }

  get showPattern(): boolean {
    return this.selectedExchange?.Type?.toLowerCase() === 'topic';
  }

  get showXMatch(): boolean {
    return this.selectedExchange?.Type?.toLowerCase() === 'headers';
  }

  get isRoutingKeyRequired(): boolean {
    return this.showRoutingKey;
  }

  get isPatternRequired(): boolean {
    return this.showPattern;
  }

  private updateFormValidation(): void {
    const routingKeyControl = this.bindingForm.get('routingKey');
    const patternControl = this.bindingForm.get('pattern');

    // Clear existing validators
    routingKeyControl?.clearValidators();
    patternControl?.clearValidators();

    // Set validators based on exchange type
    if (this.selectedExchange) {
      const exchangeType = this.selectedExchange.Type?.toLowerCase();
      
      if (exchangeType === 'direct') {
        routingKeyControl?.setValidators([Validators.required]);
      } else if (exchangeType === 'topic') {
        patternControl?.setValidators([Validators.required]);
      }
    }

    routingKeyControl?.updateValueAndValidity();
    patternControl?.updateValueAndValidity();
  }

  // Modal and CRUD operations
  openCreateModal(): void {
    this.createModalVisible = true;
    this.resetForm();
    this.showAlert = false;
  }

  private resetForm(): void {
    // Reset all models
    this.selectedVNamespace = null;
    this.selectedExchange = null;
    this.selectedQueue = null;
    
    // Reset form
    this.bindingForm.reset();
    
    // Set default values
    this.bindingForm.patchValue({
      bindingType: 'classic',
      xMatch: 'all'
    });
    
    // Disable dependent controls
    this.exchangeCtrl.disable();
    this.queueCtrl.disable();
  }

  createBinding(): void {
    // Validar que todos los modelos requeridos estén seleccionados y tengan valores válidos
    const isValidData = this.validateSelectedModels();
    
    // Validación alternativa usando FormControls como respaldo
    const vnamespaceValue = this.selectedVNamespace || this.vnamespaceCtrl.value;
    const exchangeValue = this.selectedExchange || this.exchangeCtrl.value;
    const queueValue = this.selectedQueue || this.queueCtrl.value;
    
    const hasValidValues = !!(
      (vnamespaceValue?.Code || vnamespaceValue?.Name) && 
      exchangeValue?.Code && 
      queueValue?.Code
    );

    console.log('Alternative validation:', {
      vnamespaceValue,
      exchangeValue,
      queueValue,
      hasValidValues,
      vnamespaceCodeOrName: vnamespaceValue?.Code || vnamespaceValue?.Name
    });
    
    if (this.bindingForm.valid && (isValidData || hasValidValues)) {
      const vnamespace = this.selectedVNamespace || this.vnamespaceCtrl.value;
      const exchange = this.selectedExchange || this.exchangeCtrl.value;
      const queue = this.selectedQueue || this.queueCtrl.value;
      
      const bindingData = {
        exchangeCode: exchange?.Code,
        queueCode: queue?.Code,
        vnamespace: vnamespace?.Code || vnamespace?.Name, // Usar Name como fallback si Code no existe
        routingKey: this.bindingForm.get('routingKey')?.value || '',
        pattern: this.bindingForm.get('pattern')?.value || '',
        xMatch: this.bindingForm.get('xMatch')?.value || 'all',
        bindingType: this.bindingForm.get('bindingType')?.value || 'classic'
      };

      console.log('Creating binding with data:', bindingData);

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
      console.log('Form validation failed:', {
        formValid: this.bindingForm.valid,
        isValidData,
        hasValidValues,
        selectedVNamespace: this.selectedVNamespace,
        selectedExchange: this.selectedExchange,
        selectedQueue: this.selectedQueue,
        vnamespaceCtrlValue: this.vnamespaceCtrl.value,
        exchangeCtrlValue: this.exchangeCtrl.value,
        queueCtrlValue: this.queueCtrl.value,
        formErrors: this.bindingForm.errors
      });
      this.bindingForm.markAllAsTouched();
      
      if (!isValidData && !hasValidValues) {
        this.showAlert = true;
        this.errorMessage = 'Por favor selecciona un VNamespace, Exchange y Queue válidos antes de crear el binding.';
      }
    }
  }

  private validateSelectedModels(): boolean {
    console.log('=== DETAILED MODEL VALIDATION ===');
    console.log('selectedVNamespace:', this.selectedVNamespace);
    console.log('selectedVNamespace?.Code:', this.selectedVNamespace?.Code);
    console.log('selectedVNamespace?.Name:', this.selectedVNamespace?.Name);
    console.log('selectedExchange:', this.selectedExchange);
    console.log('selectedExchange?.Code:', this.selectedExchange?.Code);
    console.log('selectedQueue:', this.selectedQueue);
    console.log('selectedQueue?.Code:', this.selectedQueue?.Code);
    
    // También verifiquemos los valores de los FormControls
    console.log('vnamespaceCtrl.value:', this.vnamespaceCtrl.value);
    console.log('exchangeCtrl.value:', this.exchangeCtrl.value);
    console.log('queueCtrl.value:', this.queueCtrl.value);

    // Para VNamespace, usar Code o Name como fallback
    const hasVNamespace = !!(this.selectedVNamespace && (this.selectedVNamespace.Code || this.selectedVNamespace.Name));
    const hasExchange = !!(this.selectedExchange && this.selectedExchange.Code);
    const hasQueue = !!(this.selectedQueue && this.selectedQueue.Code);
    
    console.log('Validation checks:', {
      hasVNamespace,
      hasExchange,
      hasQueue
    });

    const isValid = hasVNamespace && hasExchange && hasQueue;
    
    console.log('Final validation result:', isValid);
    console.log('=== END MODEL VALIDATION ===');

    return isValid;
  }

  loadBindings(cursor: string = '', isPrevious: boolean = false): void {
    if (!isPrevious && cursor) {
      this.cursors.push(cursor);
    }
    
    const vnamespaceFilter = this.selectedVNamespaceFilter?.Code || '';
    
    this.bindingsService.getBindings(this.tenantId, cursor, this.pageSize, this.searchQuery, vnamespaceFilter).subscribe({
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

  deleteBinding(): void {
    if (this.selectedBinding) {
      this.bindingsService.deleteBinding(
        this.tenantId, 
        this.selectedBinding.ExchangeCode,
        this.selectedBinding.QueueCode,
        this.selectedBinding.VNamespace
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

  getBindingTypeColor(type?: string): string {
    const typeColors: { [key: string]: string } = {
      'classic': 'primary',
      'dynamic': 'success'
    };
    return typeColors[type || 'classic'] || 'secondary';
  }

  getXMatchColor(xMatch?: string): string {
    const xMatchColors: { [key: string]: string } = {
      'all': 'info',
      'any': 'warning'
    };
    return xMatchColors[xMatch || 'all'] || 'secondary';
  }

  // Public method for template
  onVNamespaceFilterChange(event?: any): void {
    this.applyFilters();
  }

  // Method to check if Exchange is disabled
  get exchangeDisabled(): boolean {
    return !this.canSelectExchange;
  }

  // Method to check if Queue is disabled  
  get queueDisabled(): boolean {
    return !this.canSelectQueue;
  }

  // Method for vnamespace selection event
  onVNamespaceSelected(event: MatAutocompleteSelectedEvent): void {
    console.log('VNamespace selected from autocomplete:', event.option.value);
    this.selectedVNamespace = event.option.value;
    console.log('selectedVNamespace after assignment:', this.selectedVNamespace);
    this.onVNamespaceChange();
  }

  // Method for queue selection event
  onQueueSelected(event: MatAutocompleteSelectedEvent): void {
    console.log('Queue selected from autocomplete:', event.option.value);
    this.selectedQueue = event.option.value;
    console.log('selectedQueue after assignment:', this.selectedQueue);
    this.onQueueChange();
  }

  // Method for exchange selection event
  onExchangeSelected(event: MatAutocompleteSelectedEvent): void {
    console.log('Exchange selected from autocomplete:', event.option.value);
    this.selectedExchange = event.option.value;
    console.log('selectedExchange after assignment:', this.selectedExchange);
    this.onExchangeChange();
  }

  // Display function for exchanges used in template
  displayExchangeFn = (exchange: Exchange): string => {
    return this.displayExchange(exchange);
  }

  // Method to check if field is required (used in template)
  isFieldRequired(fieldName: string): boolean {
    if (fieldName === 'routingKey') {
      return this.isRoutingKeyRequired;
    }
    if (fieldName === 'pattern') {
      return this.isPatternRequired;
    }
    return false;
  }

  getExchangeTypeDisplayName(): string {
    if (!this.selectedExchange) return '';
    
    const type = this.selectedExchange.Type?.toLowerCase();
    switch (type) {
      case 'direct':
        return 'Direct (point-to-point routing)';
      case 'topic':
        return 'Topic (pattern-based routing)';
      case 'fanout':
        return 'Fanout (broadcast to all queues)';
      case 'headers':
        return 'Headers (attribute-based routing)';
      default:
        return type || '';
    }
  }
}
