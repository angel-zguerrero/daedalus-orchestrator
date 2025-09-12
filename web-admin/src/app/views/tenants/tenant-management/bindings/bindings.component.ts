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
  Code?: string;
  RoutingKey?: string;
  Pattern?: string;
  XMatch?: string;
  BindingType: string;
  ID?: string;
  CreatedAt?: string;
  UpdatedAt?: string;
  Headers?: { [key: string]: string }; // Added headers support
  // Virtual objects for dynamic binding resolution
  virtualExchange?: {
    exchangeType?: string;
    code?: string;
    name?: string;
  };
  virtualQueue?: {
    queueType?: string;
    code?: string;
    name?: string;
  };
  routingHeaders?: Array<{
    key: string;
    value: string;
  }>;
  // Objetos completos cuando includeObjects=true
  Exchange?: Exchange;
  Queue?: Queue;
  TargetExchange?: Exchange;
  AlternateExchange?: Exchange;
  // Compatibilidad con propiedades en camelCase
  code?: string;
  exchangeCode: string;
  queueCode: string;
  targetExchangeCode?: string;
  alternateExchangeCode?: string;
  TargetExchangeType?: string; // Added TargetExchangeType
  targetExchangeType?: string; // Added targetExchangeType in camelCase
  vnamespace: string;
  VNamespace?: string; // Possible variation
  Vnamespace?: string; // Possible variation
  routingKey?: string;
  pattern?: string;
  xMatch?: string;
  bindingType?: string;
  id?: string;
  createdAt?: string;
  updatedAt?: string;
  headers?: { [key: string]: string }; // Added headers support in camelCase
  exchange?: Exchange;
  queue?: Queue;
  targetExchange?: Exchange;
  alternateExchange?: Exchange;
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
  selectedTargetExchange: Exchange | null = null;
  selectedAlternateExchange: Exchange | null = null;

  // Filter model for list
  selectedVNamespaceFilter: VNamespace | null = null;

  bindingTypes = [
    { value: 'classic', label: 'Classic' },
    { value: 'dynamic', label: 'Dynamic' }
  ];

  targetExchangeTypes = [
    { value: 'queue', label: 'Queue' },
    { value: 'exchange', label: 'Exchange' }
  ];

  xMatchTypes = [
    { value: 'all', label: 'All' },
    { value: 'any', label: 'Any' }
  ];

  // Form Controls
  vnamespaceCtrl = new FormControl<VNamespace | null>(null, Validators.required);
  exchangeCtrl = new FormControl<Exchange | null>({ value: null, disabled: true }, Validators.required);
  queueCtrl = new FormControl<Queue | null>({ value: null, disabled: true }, Validators.required);
  targetExchangeCtrl = new FormControl<Exchange | null>({ value: null, disabled: true });
  alternateExchangeCtrl = new FormControl<Exchange | null>({ value: null, disabled: true });
  vnamespaceFilterCtrl = new FormControl<VNamespace | null>(null);

  // Observables for autocompletes
  filteredVNamespaces!: Observable<VNamespace[]>;
  filteredExchanges!: Observable<Exchange[]>;
  filteredQueues!: Observable<Queue[]>;
  filteredTargetExchanges!: Observable<Exchange[]>;
  filteredAlternateExchanges!: Observable<Exchange[]>;
  filteredFilterVNamespaces!: Observable<VNamespace[]>;

  // Loading states
  loadingVNamespaces = false;
  loadingExchanges = false;
  loadingQueues = false;

  // Headers management for Headers exchange type
  headers: { key: string; value: string }[] = [];
  newHeaderKey = '';
  newHeaderValue = '';

  constructor(
    private bindingsService: BindingsService,
    private exchangesService: ExchangesService,
    private queuesService: QueuesService,
    private vNamespacesService: VNamespacesService,
    private fb: FormBuilder,
    private cdr: ChangeDetectorRef
  ) {
    this.bindingForm = this.fb.group({
      code: ['', Validators.required],
      vnamespace: this.vnamespaceCtrl,
      exchange: this.exchangeCtrl,
      queue: this.queueCtrl,
      targetExchange: this.targetExchangeCtrl,
      alternateExchange: this.alternateExchangeCtrl,
      targetExchangeType: ['queue', Validators.required],
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

    // Target Exchange autocomplete
    this.filteredTargetExchanges = this.targetExchangeCtrl.valueChanges.pipe(
      startWith(null),
      debounceTime(300),
      switchMap(value => this._filterExchanges(this.getSearchTerm(value)))
    );

    // Alternate Exchange autocomplete
    this.filteredAlternateExchanges = this.alternateExchangeCtrl.valueChanges.pipe(
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
      this.selectedVNamespace = vnamespace;
      this.onVNamespaceChange();
    });

    // Watch Exchange changes
    this.exchangeCtrl.valueChanges.subscribe(exchange => {
      this.selectedExchange = exchange;
      this.onExchangeChange();
    });

    // Watch Queue changes
    this.queueCtrl.valueChanges.subscribe(queue => {
      this.selectedQueue = queue;
      this.onQueueChange();
    });

    // Watch Target Exchange changes
    this.targetExchangeCtrl.valueChanges.subscribe(targetExchange => {
      this.selectedTargetExchange = targetExchange;
    });

    // Watch Alternate Exchange changes
    this.alternateExchangeCtrl.valueChanges.subscribe(alternateExchange => {
      this.selectedAlternateExchange = alternateExchange;
    });

    // Watch BindingType changes
    this.bindingForm.get('bindingType')?.valueChanges.subscribe(bindingType => {
      this.onBindingTypeChange(bindingType);
    });

    // Watch TargetExchangeType changes
    this.bindingForm.get('targetExchangeType')?.valueChanges.subscribe(targetType => {
      this.onTargetExchangeTypeChange(targetType);
    });

    // Watch VNamespace filter changes
    this.vnamespaceFilterCtrl.valueChanges.subscribe(vnamespace => {
      // Si vnamespace es un string (solo el Code), crear un objeto VNamespace básico
      if (typeof vnamespace === 'string') {
        this.selectedVNamespaceFilter = {
          Code: vnamespace,
          Name: vnamespace, // Usar el mismo valor para Name si no tenemos el objeto completo
          Description: undefined
        };
      } else {
        this.selectedVNamespaceFilter = vnamespace;
      }
      
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
        
        return (response.data || []).map((item: any) => {
          // Intentar diferentes propiedades que podrían contener el código
          const mappedItem = {
            Code: item.Code || item.code || item.VirtualNamespaceCode || item.virtualNamespaceCode || item.Name || item.name,
            Name: item.Name || item.name || item.DisplayName || item.displayName || item.Code || item.code,
            Description: item.Description || item.description
          } as VNamespace;
          
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
        let exchanges = (response.result?.Entities || []).map((item: any) => ({
          Code: item.Code,
          Name: item.Name,
          Type: item.Type,
          VNamespace: item.VNamespace,
          Description: item.Description
        } as Exchange));

        // Filter out Fanout exchanges for dynamic bindings
        const bindingType = this.bindingForm.get('bindingType')?.value;
        if (bindingType === 'dynamic') {
          exchanges = exchanges.filter((exchange: Exchange) => exchange.Type !== 'fanout');
        }

        return exchanges;
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
          Description: item.Description,
          Type: item.Type || item.type || 'standard' // Agregar el Type
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
    // Reset dependent selections
    this.selectedExchange = null;
    this.selectedQueue = null;
    this.selectedTargetExchange = null;
    this.selectedAlternateExchange = null;
    this.exchangeCtrl.setValue(null);
    this.queueCtrl.setValue(null);
    this.targetExchangeCtrl.setValue(null);
    this.alternateExchangeCtrl.setValue(null);
    
    // Enable/disable controls based on vnamespace selection
    if (this.selectedVNamespace) {
      this.exchangeCtrl.enable();
      this.queueCtrl.enable();
      this.alternateExchangeCtrl.enable();
      
      // Enable target exchange only if target type is exchange
      const targetType = this.bindingForm.get('targetExchangeType')?.value;
      if (targetType === 'exchange') {
        this.targetExchangeCtrl.enable();
      }
    } else {
      this.exchangeCtrl.disable();
      this.queueCtrl.disable();
      this.targetExchangeCtrl.disable();
      this.alternateExchangeCtrl.disable();
    }
    
    this.updateFormValidation();
    this.cdr.detectChanges();
  }

  private onExchangeChange(): void {
    this.updateFormValidation();
    this.cdr.detectChanges();
  }

  private onQueueChange(): void {
    // Queue change logic if needed
  }

  onBindingTypeChange(event: any): void {
    if(!event){ return; }
    const bindingType = event.target?.value || event;
    // Clear queue selection when switching to dynamic binding
    if (bindingType === 'dynamic') {
      this.selectedQueue = null;
      this.queueCtrl.setValue(null);
      this.queueCtrl.disable();
      
      // Clear exchange selection if it's a Fanout exchange (not allowed for dynamic bindings)
      if (this.selectedExchange && this.selectedExchange.Type === 'fanout') {
        this.selectedExchange = null;
        this.exchangeCtrl.setValue(null);
      }
    } else if (bindingType === 'classic') {
      this.queueCtrl.enable();
    }
    
    // Trigger exchange list refresh to apply filtering
    if (this.selectedVNamespace) {
      this.exchangeCtrl.updateValueAndValidity();
    }
    
    this.updateFormValidation();
    this.cdr.detectChanges();
  }

  onTargetExchangeTypeChange(targetType: string): void {
    // When target type changes, clear current selections and update validation
    if (targetType === 'queue') {
      // Clear target exchange selection
      this.selectedTargetExchange = null;
      this.targetExchangeCtrl.setValue(null);
      this.targetExchangeCtrl.disable();
      
      // Enable queue control if it was disabled
      if (this.bindingForm.get('bindingType')?.value === 'classic') {
        this.queueCtrl.enable();
      }
    } else if (targetType === 'exchange') {
      // Clear queue selection
      this.selectedQueue = null;
      this.queueCtrl.setValue(null);
      this.queueCtrl.disable();
      
      // Enable target exchange control
      this.targetExchangeCtrl.enable();
    }
    
    this.updateFormValidation();
    this.cdr.detectChanges();
  }

  // Display functions for autocompletes
  displayVNamespace = (vnamespace: VNamespace): string => {
    return vnamespace ? `${vnamespace.Code}` : '';
  }

  displayExchange = (exchange: Exchange): string => {
    return exchange ? `${exchange.Code} - ${exchange.Name}` : '';
  }

  displayQueue = (queue: Queue): string => {
    return queue ? `${queue.Code} - ${queue.Name}` : '';
  }

  displayTargetExchange = (exchange: Exchange): string => {
    return exchange ? `${exchange.Code} - ${exchange.Name} (${exchange.Type})` : '';
  }

  displayAlternateExchange = (exchange: Exchange): string => {
    return exchange ? `${exchange.Code} - ${exchange.Name} (${exchange.Type})` : '';
  }

  // Validation and visibility methods
  get canSelectExchange(): boolean {
    return this.selectedVNamespace !== null;
  }

  get canSelectQueue(): boolean {
    return this.selectedVNamespace !== null;
  }

  get showQueue(): boolean {
    const bindingType = this.bindingForm.get('bindingType')?.value;
    const targetType = this.bindingForm.get('targetExchangeType')?.value;
    return bindingType === 'classic' && targetType === 'queue';
  }

  get showRoutingKey(): boolean {
    const bindingType = this.bindingForm.get('bindingType')?.value;
    // Don't show routing key for dynamic bindings as it's ignored
    return this.selectedExchange?.Type?.toLowerCase() === 'direct' && bindingType !== 'dynamic';
  }

  get showPattern(): boolean {
    const bindingType = this.bindingForm.get('bindingType')?.value;
    const isTopicExchange = this.selectedExchange?.Type?.toLowerCase() === 'topic';
    
    // Show pattern for topic exchanges regardless of binding type
    // For dynamic bindings with topic exchanges, pattern is used to find queues automatically
    return isTopicExchange;
  }

  get showXMatch(): boolean {
    return this.selectedExchange?.Type?.toLowerCase() === 'headers';
  }

  get showHeaders(): boolean {
    // Headers are only shown for Headers exchanges with classic binding type
    // For dynamic bindings, queues are determined automatically based on message headers
    return this.selectedExchange?.Type?.toLowerCase() === 'headers' && 
           this.bindingForm.get('bindingType')?.value === 'classic';
  }

  get showTargetExchange(): boolean {
    const bindingType = this.bindingForm.get('bindingType')?.value;
    const targetType = this.bindingForm.get('targetExchangeType')?.value;
    return bindingType === 'classic' && targetType === 'exchange' && this.selectedVNamespace !== null;
  }

  get showAlternateExchange(): boolean {
    // Always show alternate exchange if a VNamespace is selected
    return this.selectedVNamespace !== null;
  }

  get isRoutingKeyRequired(): boolean {
    return this.showRoutingKey;
  }

  get isPatternRequired(): boolean {
    return this.showPattern;
  }

  get isQueueRequired(): boolean {
    return this.showQueue;
  }

  private updateFormValidation(): void {
    const routingKeyControl = this.bindingForm.get('routingKey');
    const patternControl = this.bindingForm.get('pattern');
    const queueControl = this.bindingForm.get('queue');

    // Clear existing validators
    routingKeyControl?.clearValidators();
    patternControl?.clearValidators();
    queueControl?.clearValidators();

    // Set validators based on binding type
    const bindingType = this.bindingForm.get('bindingType')?.value;
    if (bindingType === 'classic') {
      queueControl?.setValidators([Validators.required]);
    }

    // Set validators based on exchange type and binding type
    if (this.selectedExchange) {
      const exchangeType = this.selectedExchange.Type?.toLowerCase();
      
      if (exchangeType === 'direct') {
        // For dynamic bindings, routing key is not required as queue is found by code
        if (bindingType === 'classic') {
          routingKeyControl?.setValidators([Validators.required]);
        }
      } else if (exchangeType === 'topic') {
        // Pattern is always required for topic exchanges, regardless of binding type
        // For dynamic bindings, pattern is used to find queues automatically
        patternControl?.setValidators([Validators.required]);
      }
    }

    routingKeyControl?.updateValueAndValidity();
    patternControl?.updateValueAndValidity();
    queueControl?.updateValueAndValidity();
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
    this.selectedTargetExchange = null;
    this.selectedAlternateExchange = null;
    
    // Reset form
    this.bindingForm.reset();
    
    // Set default values
    this.bindingForm.patchValue({
      bindingType: 'classic',
      targetExchangeType: 'queue',
      xMatch: 'all'
    });
    
    // Reset headers
    this.headers = [];
    this.newHeaderKey = '';
    this.newHeaderValue = '';
    
    // Disable dependent controls
    this.exchangeCtrl.disable();
    this.queueCtrl.disable();
    this.targetExchangeCtrl.disable();
    this.alternateExchangeCtrl.disable();
  }

  createBinding(): void {
    // Validar que todos los modelos requeridos estén seleccionados y tengan valores válidos
    const isValidData = this.validateSelectedModels();
    
    // Validación alternativa usando FormControls como respaldo
    const vnamespaceValue = this.selectedVNamespace || this.vnamespaceCtrl.value;
    const exchangeValue = this.selectedExchange || this.exchangeCtrl.value;
    const queueValue = this.selectedQueue || this.queueCtrl.value;
    const targetExchangeValue = this.selectedTargetExchange || this.targetExchangeCtrl.value;
    const bindingType = this.bindingForm.get('bindingType')?.value || 'classic';
    const targetExchangeType = this.bindingForm.get('targetExchangeType')?.value || 'queue';
    
    const hasValidValues = !!(
      (vnamespaceValue?.Code || vnamespaceValue?.Name) && 
      exchangeValue?.Code && 
      (
        bindingType === 'dynamic' || 
        (targetExchangeType === 'queue' && queueValue?.Code) ||
        (targetExchangeType === 'exchange' && targetExchangeValue?.Code)
      )
    );
    
    if (this.bindingForm.valid && (isValidData || hasValidValues)) {
      const vnamespace = this.selectedVNamespace || this.vnamespaceCtrl.value;
      const exchange = this.selectedExchange || this.exchangeCtrl.value;
      const queue = this.selectedQueue || this.queueCtrl.value;
      const targetExchange = this.selectedTargetExchange || this.targetExchangeCtrl.value;
      const alternateExchange = this.selectedAlternateExchange || this.alternateExchangeCtrl.value;
      const targetExchangeType = this.bindingForm.get('targetExchangeType')?.value || 'queue';
      
      const bindingData = {
        code: this.bindingForm.get('code')?.value,
        exchangeCode: exchange?.Code,
        queueCode: targetExchangeType === 'queue' ? (queue?.Code || '') : '', // Only use queue if target type is queue
        targetExchangeCode: targetExchangeType === 'exchange' ? (targetExchange?.Code || '') : '', // Only use target exchange if target type is exchange
        alternateExchangeCode: alternateExchange?.Code || '',
        vnamespace: vnamespace?.Code || vnamespace?.Name, // Usar Name como fallback si Code no existe
        routingKey: this.bindingForm.get('routingKey')?.value || '',
        pattern: this.bindingForm.get('pattern')?.value || '',
        xMatch: this.bindingForm.get('xMatch')?.value || 'all',
        bindingType: bindingType,
        targetExchangeType: targetExchangeType,
        headers: this.showHeaders ? this.getHeadersAsMap() : {}
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
      
      if (!isValidData && !hasValidValues) {
        this.showAlert = true;
        if (bindingType === 'classic') {
          this.errorMessage = 'Por favor selecciona un VNamespace, Exchange y Queue válidos antes de crear el binding.';
        } else {
          this.errorMessage = 'Por favor selecciona un VNamespace y Exchange válidos antes de crear el binding dinámico.';
        }
      }
    }
  }

  private validateSelectedModels(): boolean {
    // Para VNamespace, usar Code o Name como fallback
    const hasVNamespace = !!(this.selectedVNamespace && (this.selectedVNamespace.Code || this.selectedVNamespace.Name));
    const hasExchange = !!(this.selectedExchange && this.selectedExchange.Code);
    const bindingType = this.bindingForm.get('bindingType')?.value;
    const targetExchangeType = this.bindingForm.get('targetExchangeType')?.value;
    
    // For classic bindings, check target type
    if (bindingType === 'classic') {
      if (targetExchangeType === 'queue') {
        // If target is queue, queue is required
        const hasQueue = !!(this.selectedQueue && this.selectedQueue.Code);
        return hasVNamespace && hasExchange && hasQueue;
      } else if (targetExchangeType === 'exchange') {
        // If target is exchange, target exchange is required
        const hasTargetExchange = !!(this.selectedTargetExchange && this.selectedTargetExchange.Code);
        return hasVNamespace && hasExchange && hasTargetExchange;
      }
      return false;
    } else {
      // For dynamic bindings, target type is not relevant
      return hasVNamespace && hasExchange;
    }
  }

  loadBindings(cursor: string = '', isPrevious: boolean = false): void {
    if (!isPrevious && cursor) {
      this.cursors.push(cursor);
    }
    
    const vnamespaceFilter = this.selectedVNamespaceFilter?.Code || this.selectedVNamespaceFilter?.Name || '';

    this.bindingsService.getBindings(this.tenantId, cursor, this.pageSize, this.searchQuery, vnamespaceFilter, true).subscribe({
      next: (response) => {
        this.bindings = response.result.Entities || [];
        this.cursor = response.result.Cursor;
        
        // Debug: Log the first binding to see the structure
        if (this.bindings.length > 0) {
          console.log('Sample binding data:', this.bindings[0]);
        }
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
    this.selectedBinding = binding;
    this.detailsModalVisible = true;
    this.showAlert = false;
  }

  private getVNamespaceFromBinding(binding: any): string {
    return binding.vnamespace || 
           binding.VNamespace || 
           binding.Vnamespace || 
           '';
  }

  // Public method for template use
  getVNamespace(binding: any): string {
    return this.getVNamespaceFromBinding(binding);
  }

  deleteBinding(): void {
    console.log('Deleting binding:::', this.selectedBinding);
    if (this.selectedBinding && this.selectedBinding.Code) {
      const vnamespace = this.getVNamespaceFromBinding(this.selectedBinding);
      
      console.log('Using vnamespace:', vnamespace);
      
      if (!vnamespace) {
        this.showAlert = true;
        this.errorMessage = 'Virtual namespace is required but not found in binding data';
        return;
      }

      this.bindingsService.deleteBinding(
        this.tenantId, 
        this.selectedBinding.Code,
        vnamespace
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
    this.selectedVNamespace = event.option.value;
    this.onVNamespaceChange();
  }

  // Method for queue selection event
  onQueueSelected(event: MatAutocompleteSelectedEvent): void {
    this.selectedQueue = event.option.value;
    this.onQueueChange();
  }

  // Method for exchange selection event
  onExchangeSelected(event: MatAutocompleteSelectedEvent): void {
    this.selectedExchange = event.option.value;
    this.onExchangeChange();
  }

  // Method for target exchange selection event
  onTargetExchangeSelected(event: MatAutocompleteSelectedEvent): void {
    this.selectedTargetExchange = event.option.value;
  }

  // Method for alternate exchange selection event
  onAlternateExchangeSelected(event: MatAutocompleteSelectedEvent): void {
    this.selectedAlternateExchange = event.option.value;
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

  // Methods for binding details modal
  getSelectedBindingExchangeType(): string {
    const exchange = this.selectedBinding?.Exchange || this.selectedBinding?.exchange;
    return exchange?.Type?.toLowerCase() || '';
  }

  shouldShowRoutingKeyInDetails(): boolean {
    return this.getSelectedBindingExchangeType() === 'direct';
  }

  shouldShowPatternInDetails(): boolean {
    return this.getSelectedBindingExchangeType() === 'topic';
  }

  shouldShowXMatchInDetails(): boolean {
    return this.getSelectedBindingExchangeType() === 'headers';
  }

  shouldShowHeadersInDetails(): boolean {
    return this.getSelectedBindingExchangeType() === 'headers';
  }

  getSelectedBindingHeaders(): { key: string; value: string }[] {
    const headers = this.selectedBinding?.Headers || this.selectedBinding?.headers;
    if (!headers) return [];
    
    return Object.entries(headers).map(([key, value]) => ({ key, value }));
  }

  getSelectedBindingExchangeTypeDisplayName(): string {
    const type = this.getSelectedBindingExchangeType();
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
        return type || 'Unknown';
    }
  }

  // Headers management methods
  addHeader(): void {
    if (this.newHeaderKey.trim() && this.newHeaderValue.trim()) {
      // Check if header key already exists
      const existingIndex = this.headers.findIndex(h => h.key === this.newHeaderKey.trim());
      if (existingIndex >= 0) {
        // Update existing header
        this.headers[existingIndex].value = this.newHeaderValue.trim();
      } else {
        // Add new header
        this.headers.push({
          key: this.newHeaderKey.trim(),
          value: this.newHeaderValue.trim()
        });
      }
      
      // Clear input fields
      this.newHeaderKey = '';
      this.newHeaderValue = '';
    }
  }

  removeHeader(index: number): void {
    if (index >= 0 && index < this.headers.length) {
      this.headers.splice(index, 1);
    }
  }

  onHeaderKeyInput(event: Event): void {
    const target = event.target as HTMLInputElement;
    this.newHeaderKey = target.value;
  }

  onHeaderValueInput(event: Event): void {
    const target = event.target as HTMLInputElement;
    this.newHeaderValue = target.value;
  }

  isHeaderKeyDuplicate(): boolean {
    return this.headers.some(h => h.key === this.newHeaderKey.trim());
  }

  private getHeadersAsMap(): { [key: string]: string } {
    const headersMap: { [key: string]: string } = {};
    this.headers.forEach(header => {
      headersMap[header.key] = header.value;
    });
    return headersMap;
  }

  getBindingTypeDisplayName(type?: string): string {
    const typeNames: { [key: string]: string } = {
      'classic': 'Classic',
      'dynamic': 'Dynamic'
    };
    return typeNames[type || 'classic'] || 'Classic';
  }

  getExchangeTypeColor(type?: string): string {
    const typeColors: { [key: string]: string } = {
      'direct': '#007bff',    // Blue
      'topic': '#28a745',     // Green
      'headers': '#dc3545',   // Red
      'fanout': '#ffc107'     // Yellow
    };
    return typeColors[type || 'direct'] || '#6c757d'; // Gray as default
  }

  // Dynamic message methods based on TargetExchangeType
  getDynamicBindingMessage(exchangeType: string): string {
    const targetType = this.bindingForm.get('targetExchangeType')?.value;
    const entityType = targetType === 'queue' ? 'Queue' : 'Exchange';
    const entityTypePlural = targetType === 'queue' ? 'Queues' : 'Exchanges';
    
    switch (exchangeType?.toLowerCase()) {
      case 'direct':
        return `Dynamic Direct Binding: ${entityType} will be determined automatically by the ${targetType} code.`;
      case 'topic':
        return `Dynamic Topic Binding: ${entityTypePlural} will be determined where the code matches the pattern.`;
      case 'headers':
        return `Dynamic Headers Binding: ${entityTypePlural} will be determined by ${targetType} headers, message headers, and X-Match type.`;
      case 'fanout':
        return `Dynamic Fanout Binding: ${entityType} will be determined automatically by the ${targetType} code.`;
      default:
        return `Dynamic Binding: ${entityTypePlural} will be determined automatically.`;
    }
  }

  getDynamicBindingDetailsMessage(exchangeType: string): string {
    const targetType = this.selectedBinding?.TargetExchangeType || this.selectedBinding?.targetExchangeType || 'queue';
    const entityType = targetType === 'queue' ? 'Queue' : 'Exchange';
    const entityTypePlural = targetType === 'queue' ? 'Queues' : 'Exchanges';
    
    switch (exchangeType?.toLowerCase()) {
      case 'direct':
        return `Dynamic Direct Binding: ${entityType} is determined automatically by the ${targetType} code.`;
      case 'topic':
        return `Dynamic Topic Binding: ${entityTypePlural} are determined where the code matches the pattern.`;
      case 'headers':
        return `Dynamic Headers Binding: ${entityTypePlural} are determined by ${targetType} headers, message headers, and X-Match type.`;
      case 'fanout':
        return `Dynamic Fanout Binding: ${entityType} is determined automatically by the ${targetType} code.`;
      default:
        return `Dynamic Binding: ${entityTypePlural} are determined automatically.`;
    }
  }
}
