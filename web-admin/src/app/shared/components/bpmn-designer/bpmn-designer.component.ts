import {
  Component,
  ElementRef,
  EventEmitter,
  Input,
  OnChanges,
  OnDestroy,
  Output,
  SimpleChanges,
  ViewChild,
  AfterViewInit,
  ViewEncapsulation
} from '@angular/core';
import { CommonModule } from '@angular/common';

import BpmnModeler from 'bpmn-js/lib/Modeler';
import BpmnNavigatedViewer from 'bpmn-js/lib/NavigatedViewer';
import {
  BpmnPropertiesPanelModule,
  BpmnPropertiesProviderModule
} from 'bpmn-js-properties-panel';
import {
  ElementTemplatesCoreModule,
  ElementTemplatesPropertiesProviderModule
} from 'bpmn-js-element-templates';
import ElementTemplateChooserModule from '@bpmn-io/element-template-chooser';
import camundaModdleDescriptor from 'camunda-bpmn-moddle/resources/camunda.json';
import elementTemplatesData from './element-templates.json';

import lintModule from 'bpmn-js-bpmnlint';
import { DesignErrorsConsoleComponent } from '../design-errors-console/design-errors-console.component';

export const DEFAULT_BPMN_XML = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL"
                  xmlns:bpmndi="http://www.omg.org/spec/BPMN/20100524/DI"
                  xmlns:dc="http://www.omg.org/spec/DD/20100524/DC"
                  xmlns:camunda="http://camunda.org/schema/1.0/bpmn"
                  id="Definitions_1"
                  targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_1" isExecutable="true">
    <bpmn:startEvent id="StartEvent_1" name="Start" />
  </bpmn:process>
  <bpmndi:BPMNDiagram id="BPMNDiagram_1">
    <bpmndi:BPMNPlane id="BPMNPlane_1" bpmnElement="Process_1">
      <bpmndi:BPMNShape id="_BPMNShape_StartEvent_2" bpmnElement="StartEvent_1">
        <dc:Bounds x="179" y="159" width="36" height="36" />
      </bpmndi:BPMNShape>
    </bpmndi:BPMNPlane>
  </bpmndi:BPMNDiagram>
</bpmn:definitions>`;

@Component({
  selector: 'app-bpmn-designer',
  templateUrl: './bpmn-designer.component.html',
  styleUrls: ['./bpmn-designer.component.scss'],
  standalone: true,
  imports: [CommonModule, DesignErrorsConsoleComponent],
  encapsulation: ViewEncapsulation.None
})
export class BpmnDesignerComponent implements AfterViewInit, OnChanges, OnDestroy {
  @ViewChild('canvasRef', { static: true }) private canvasRef!: ElementRef<HTMLDivElement>;
  @ViewChild('propertiesRef', { static: true }) private propertiesRef!: ElementRef<HTMLDivElement>;

  @Input() payload: string = '';
  @Input() readonly: boolean = false;
  @Output() xmlChange = new EventEmitter<string>();
  @Output() designErrorsChange = new EventEmitter<{ hasErrors: boolean; errors: string[] }>();

  public currentLintErrors: string[] = [];
  public hasLintErrors: boolean = false;

  private bpmnModeler!: any;
  private isInitialized = false;
  private lastEmittedXml: string = '';
  private constraintObserver?: MutationObserver;

  ngAfterViewInit(): void {
    this.initModeler();
  }

  ngOnChanges(changes: SimpleChanges): void {
    if (changes['readonly'] && this.isInitialized && !changes['readonly'].firstChange) {
      this.reinitModeler();
    } else if (changes['payload'] && this.isInitialized && !changes['payload'].firstChange) {
      const newPayload = (this.payload || '').trim();
      if (newPayload !== this.lastEmittedXml.trim()) {
        this.importXml(newPayload || DEFAULT_BPMN_XML);
      }
    }
  }

  ngOnDestroy(): void {
    if (this.constraintObserver) {
      this.constraintObserver.disconnect();
      this.constraintObserver = undefined;
    }
    if (this.bpmnModeler) {
      this.bpmnModeler.destroy();
    }
  }

  public reinitModeler(): void {
    if (this.bpmnModeler) {
      try {
        this.bpmnModeler.destroy();
      } catch (e) {
        // ignore destroy error
      }
      this.bpmnModeler = null;
    }
    this.isInitialized = false;
    setTimeout(() => {
      this.initModeler();
    }, 50);
  }

  private initModeler(): void {
    if (!this.canvasRef || !this.canvasRef.nativeElement) {
      console.warn('Canvas element reference is not ready yet');
      return;
    }

    if (this.readonly) {
      this.bpmnModeler = new BpmnNavigatedViewer({
        container: this.canvasRef.nativeElement
      });
    } else {
      const propertiesParent = this.propertiesRef?.nativeElement;
      const modelerConfig: any = {
        container: this.canvasRef.nativeElement,
        additionalModules: [
          BpmnPropertiesPanelModule,
          BpmnPropertiesProviderModule,
          ElementTemplatesCoreModule,
          ElementTemplatesPropertiesProviderModule,
          ElementTemplateChooserModule,
          lintModule
        ],
        moddleExtensions: {
          camunda: camundaModdleDescriptor
        },
        elementTemplates: elementTemplatesData
      };

      if (propertiesParent) {
        modelerConfig.propertiesPanel = {
          parent: propertiesParent
        };
      }

      this.bpmnModeler = new BpmnModeler(modelerConfig);

      try {
        const paletteProvider = this.bpmnModeler.get('paletteProvider');
        if (paletteProvider && paletteProvider.getPaletteEntries) {
          const origGetPaletteEntries = paletteProvider.getPaletteEntries.bind(paletteProvider);
          paletteProvider.getPaletteEntries = function(element: any) {
            const entries = origGetPaletteEntries(element);
            delete entries['create.subprocess-expanded'];
            delete entries['create.data-object'];
            delete entries['create.data-store'];
            delete entries['create.participant-expanded'];
            delete entries['create.group'];
            return entries;
          };
        }
      } catch (e) {
        console.warn('Palette entries override notice:', e);
      }

      try {
        const contextPadProvider = this.bpmnModeler.get('contextPadProvider');
        if (contextPadProvider && contextPadProvider.getContextPadEntries) {
          const origGetContextPadEntries = contextPadProvider.getContextPadEntries.bind(contextPadProvider);
          contextPadProvider.getContextPadEntries = function(element: any) {
            const entries = origGetContextPadEntries(element);
            const elementType = element && element.type ? element.type.toLowerCase() : '';
            const isGateway = elementType.includes('gateway');
            const isFlow = elementType.includes('flow') || elementType.includes('sequence') || elementType.includes('association');
            if (!isGateway && !isFlow) {
              delete entries['replace'];
            }
            return entries;
          };
        }
      } catch (e) {
        console.warn('Context pad entries override notice:', e);
      }

      try {
        const replaceMenuProvider = this.bpmnModeler.get('replaceMenuProvider');
        if (replaceMenuProvider && replaceMenuProvider.getEntries) {
          const origGetEntries = replaceMenuProvider.getEntries.bind(replaceMenuProvider);
          replaceMenuProvider.getEntries = function(element: any) {
            const entries = origGetEntries(element);
            if (Array.isArray(entries)) {
              return entries.filter((e: any) => {
                const id = (e.id || e.actionName || '').toLowerCase();
                const label = (e.label || e.name || '').toLowerCase();
                return !id.includes('complex') && !id.includes('event-based') &&
                       !label.includes('complex') && !label.includes('event-based');
              });
            } else if (entries && typeof entries === 'object') {
              delete entries['replace-with-complex-gateway'];
              delete entries['replace-with-event-based-gateway'];
              return entries;
            }
            return entries;
          };
        }
      } catch (e) {
        console.warn('Replace menu provider override notice:', e);
      }

      try {
        const elementTemplates = this.bpmnModeler.get('elementTemplates');
        if (elementTemplates && elementTemplates.set) {
          elementTemplates.set(elementTemplatesData);
        }
        const loader = this.bpmnModeler.get('elementTemplatesLoader');
        if (loader && loader.setTemplates) {
          loader.setTemplates(elementTemplatesData);
        }
      } catch (e) {
        console.warn('Element templates loader notice:', e);
      }

      this.setupPropertiesPanelConstraintEnhancer();
    }

    this.isInitialized = true;

    // Listen for diagram changes to emit xmlChange and run lint validation
    if (!this.readonly && this.bpmnModeler) {
      const eventBus = this.bpmnModeler.get('eventBus');
      if (eventBus) {
        eventBus.on('commandStack.changed', () => {
          this.emitCurrentXml();
          this.runLintValidation();
        });
        eventBus.on('import.done', () => {
          setTimeout(() => this.runLintValidation(), 150);
        });
      }
    }

    const initialXml = this.payload && this.payload.trim().startsWith('<?xml')
      ? this.payload
      : DEFAULT_BPMN_XML;

    this.importXml(initialXml);
  }

  public runLintValidation(): { hasErrors: boolean; errors: string[] } {
    if (!this.bpmnModeler || this.readonly) {
      this.currentLintErrors = [];
      this.hasLintErrors = false;
      return { hasErrors: false, errors: [] };
    }

    const errors: string[] = [];

    const getGatewayTypeName = (typeStr: string): string => {
      const cleanType = (typeStr || '').replace(/^bpmn:/, '');
      switch (cleanType) {
        case 'ExclusiveGateway': return 'Exclusive Gateway (XOR)';
        case 'ParallelGateway': return 'Parallel Gateway (AND)';
        case 'InclusiveGateway': return 'Inclusive Gateway (OR)';
        case 'ComplexGateway': return 'Complex Gateway';
        case 'EventBasedGateway': return 'Event-Based Gateway';
        default: return cleanType;
      }
    };

    try {
      const elementRegistry = this.bpmnModeler.get('elementRegistry');
      if (elementRegistry) {
        const elements = elementRegistry.getAll();
        let startEventCount = 0;
        let endEventCount = 0;
        const reportedGatewayPairs = new Set<string>();

        const IGNORED_TYPES = [
          'label',
          'bpmn:Process',
          'bpmn:Collaboration',
          'bpmn:Participant',
          'bpmn:Lane',
          'bpmn:TextAnnotation',
          'bpmn:Group',
          'bpmn:DataStoreReference',
          'bpmn:DataObjectReference'
        ];

        elements.forEach((el: any) => {
          if (!el || !el.type || IGNORED_TYPES.includes(el.type)) {
            return;
          }

          const rawType = el.type || '';
          const type = rawType.replace(/^bpmn:/, '');
          const id = el.id || '';
          const rawName = (el.businessObject?.name || '').trim();
          const displayName = rawName ? `'${rawName}' (${id})` : `'${id}'`;

          // Check if element is a connection / edge (SequenceFlow, Association, MessageFlow, etc.)
          const isConnection = !!el.waypoints || type === 'SequenceFlow' || type === 'Association' || type === 'MessageFlow';

          if (isConnection) {
            // A sequence flow connection is disconnected ONLY IF missing source or target
            if (!el.source || !el.target) {
              errors.push(`Sequence Flow ${displayName} is disconnected (missing source or target).`);
            }
            return; // Connections are edges, not nodes; skip shape incoming/outgoing checks!
          }

          // Shape / Node checks
          const incoming = el.incoming || [];
          const outgoing = el.outgoing || [];

          if (type === 'StartEvent') {
            startEventCount++;
            if (outgoing.length === 0) {
              errors.push(`Start Event ${displayName} has no outgoing connection.`);
            }
          } else if (type === 'EndEvent') {
            endEventCount++;
            if (incoming.length === 0) {
              errors.push(`End Event ${displayName} has no incoming connection.`);
            }
          } else if (type === 'BoundaryEvent') {
            if (outgoing.length === 0) {
              errors.push(`Boundary Event ${displayName} has no outgoing connection.`);
            }
          } else {
            // Regular flow nodes (Tasks, Gateways, SubProcesses, etc.)
            if (incoming.length === 0 && outgoing.length === 0) {
              errors.push(`Element ${displayName} (${type}) is disconnected (has no connections).`);
            } else if (incoming.length === 0) {
              errors.push(`Element ${displayName} (${type}) has no incoming connection.`);
            } else if (outgoing.length === 0) {
              errors.push(`Element ${displayName} (${type}) has no outgoing connection.`);
            }
          }

          // Gateway checks:
          if (type.includes('Gateway') && outgoing.length > 1) {
            const divType = type;
            const defaultFlowId = el.businessObject?.default?.id;

            // 1. Condition check on outgoing flows (ONLY for conditional divergent gateways: Exclusive, Inclusive, etc. Tasks & Parallel Gateways NEVER require conditions)
            const isConditionalGateway = divType !== 'ParallelGateway';
            if (isConditionalGateway) {
              outgoing.forEach((flow: any) => {
                const cond = (flow.businessObject?.conditionExpression?.body || '').trim();
                const isDefault = defaultFlowId && flow.id === defaultFlowId;
                if (!cond && !isDefault) {
                  const targetId = flow.target?.id || 'target';
                  const targetRawName = (flow.target?.businessObject?.name || '').trim();
                  const targetDisplayName = targetRawName ? `'${targetRawName}' (${targetId})` : `'${targetId}'`;
                  errors.push(`Gateway ${displayName} flow to ${targetDisplayName} has no condition and is not marked as default flow.`);
                }
              });
            }

            // 2. Nesting-aware Gateway Symmetry & Deadlock Check (Simetría de Compuertas Anidadas)
            interface QueueItem {
              node: any;
              stack: string[];
            }
            const queue: QueueItem[] = [];
            const visitedState = new Set<string>();

            outgoing.forEach((flow: any) => {
              if (flow && flow.target) {
                queue.push({
                  node: flow.target,
                  stack: [id]
                });
              }
            });

            while (queue.length > 0) {
              const item = queue.shift();
              if (!item) continue;
              const { node, stack } = item;
              if (!node || !node.id) continue;

              const stateKey = `${node.id}:${stack.join(',')}`;
              if (visitedState.has(stateKey)) {
                continue;
              }
              visitedState.add(stateKey);

              const nodeRawType = node.type || '';
              const nodeType = nodeRawType.replace(/^bpmn:/, '');
              const nodeIncomingCount = (node.incoming || []).length;
              const nodeOutgoingCount = (node.outgoing || []).length;

              let currentStack = [...stack];

              if (nodeType.includes('Gateway') && nodeIncomingCount > 1) {
                if (currentStack.length > 0) {
                  const topDivId = currentStack[currentStack.length - 1];
                  const topDivNode = elements.find((e: any) => e.id === topDivId);

                  if (topDivNode) {
                    const topDivRawType = topDivNode.type || '';
                    const topDivType = topDivRawType.replace(/^bpmn:/, '');
                    const convType = nodeType;
                    const convId = node.id;

                    const pairKey = `${topDivId}->${convId}`;
                    if (!reportedGatewayPairs.has(pairKey)) {
                      if (topDivType !== convType) {
                        reportedGatewayPairs.add(pairKey);

                        const topDivRawName = (topDivNode.businessObject?.name || '').trim();
                        const topDivDisplayName = topDivRawName ? `'${topDivRawName}' (${topDivId})` : `'${topDivId}'`;
                        const convRawName = (node.businessObject?.name || '').trim();
                        const convDisplayName = convRawName ? `'${convRawName}' (${convId})` : `'${convId}'`;
                        const divTypeName = getGatewayTypeName(topDivType);
                        const convTypeName = getGatewayTypeName(convType);

                        if (topDivType === 'ExclusiveGateway' && convType === 'ParallelGateway') {
                          errors.push(
                            `Deadlock Error: ${divTypeName} ${topDivDisplayName} diverges into ${convTypeName} ${convDisplayName}. A Parallel Gateway waits for all incoming branches, causing a permanent deadlock because an Exclusive Gateway only activates one branch.`
                          );
                        } else if (topDivType === 'ExclusiveGateway' && convType === 'InclusiveGateway') {
                          errors.push(
                            `Gateway Asymmetry Error: ${divTypeName} ${topDivDisplayName} diverges into ${convTypeName} ${convDisplayName}. Gateways must be symmetric (Exclusive Gateway divergence must converge with an Exclusive Gateway).`
                          );
                        } else {
                          errors.push(
                            `Gateway Asymmetry Error: ${divTypeName} ${topDivDisplayName} diverges into ${convTypeName} ${convDisplayName}. Gateways must be symmetric (divergence and convergence must use the same gateway type).`
                          );
                        }
                      }
                    }
                  }

                  currentStack.pop();
                }
              }

              if (nodeType.includes('Gateway') && nodeOutgoingCount > 1 && node.id !== id) {
                currentStack.push(node.id);
              }

              (node.outgoing || []).forEach((flow: any) => {
                if (flow && flow.target) {
                  queue.push({
                    node: flow.target,
                    stack: [...currentStack]
                  });
                }
              });
            }
          }
        });

        if (startEventCount === 0) {
          errors.push('Process must contain at least one Start Event.');
        }
        if (endEventCount === 0) {
          errors.push('Process must contain at least one End Event.');
        }
      }
    } catch (e) {
      console.warn('Lint validation notice:', e);
    }

    this.currentLintErrors = errors;
    this.hasLintErrors = errors.length > 0;
    const result = { hasErrors: this.hasLintErrors, errors: this.currentLintErrors };
    this.designErrorsChange.emit(result);
    return result;
  }

  public refresh(): void {
    if (!this.bpmnModeler) return;
    try {
      const canvas = this.bpmnModeler.get('canvas');
      canvas.resized();
      canvas.zoom('fit-viewport');
    } catch {
      // ignore resize error
    }
  }

  public async importXml(xml: string): Promise<void> {
    if (!this.bpmnModeler) return;
    try {
      const xmlToLoad = xml && xml.trim().startsWith('<?xml') ? xml : DEFAULT_BPMN_XML;
      this.lastEmittedXml = xmlToLoad.trim();
      await this.bpmnModeler.importXML(xmlToLoad);
      setTimeout(() => {
        this.refresh();
      }, 100);
    } catch (err) {
      console.error('Failed to import BPMN XML diagram:', err);
    }
  }

  public async getXml(): Promise<string> {
    if (!this.bpmnModeler) return DEFAULT_BPMN_XML;
    try {
      const result = await this.bpmnModeler.saveXML({ format: true });
      return result.xml || '';
    } catch (err) {
      console.error('Failed to save BPMN XML:', err);
      return '';
    }
  }

  private async emitCurrentXml(): Promise<void> {
    const xml = await this.getXml();
    this.lastEmittedXml = (xml || '').trim();
    this.xmlChange.emit(xml);
  }

  public zoomIn(): void {
    if (this.bpmnModeler) {
      try {
        const canvas = this.bpmnModeler.get('canvas');
        const currentZoom = canvas.zoom() || 1;
        canvas.zoom(currentZoom * 1.2);
      } catch (err) {
        console.error('Zoom in failed:', err);
      }
    }
  }

  public zoomOut(): void {
    if (this.bpmnModeler) {
      try {
        const canvas = this.bpmnModeler.get('canvas');
        const currentZoom = canvas.zoom() || 1;
        canvas.zoom(currentZoom / 1.2);
      } catch (err) {
        console.error('Zoom out failed:', err);
      }
    }
  }

  public zoomReset(): void {
    if (this.bpmnModeler) {
      try {
        const canvas = this.bpmnModeler.get('canvas');
        canvas.zoom('fit-viewport');
      } catch (err) {
        console.error('Zoom reset failed:', err);
      }
    }
  }

  public undo(): void {
    if (this.bpmnModeler && !this.readonly) {
      try {
        const commandStack = this.bpmnModeler.get('commandStack');
        if (commandStack && commandStack.canUndo()) {
          commandStack.undo();
        }
      } catch (err) {
        console.error('Undo action failed:', err);
      }
    }
  }

  public redo(): void {
    if (this.bpmnModeler && !this.readonly) {
      try {
        const commandStack = this.bpmnModeler.get('commandStack');
        if (commandStack && commandStack.canRedo()) {
          commandStack.redo();
        }
      } catch (err) {
        console.error('Redo action failed:', err);
      }
    }
  }

  public canUndo(): boolean {
    if (!this.bpmnModeler || this.readonly) return false;
    try {
      const commandStack = this.bpmnModeler.get('commandStack');
      return commandStack ? commandStack.canUndo() : false;
    } catch {
      return false;
    }
  }

  public canRedo(): boolean {
    if (!this.bpmnModeler || this.readonly) return false;
    try {
      const commandStack = this.bpmnModeler.get('commandStack');
      return commandStack ? commandStack.canRedo() : false;
    } catch {
      return false;
    }
  }

  private setupPropertiesPanelConstraintEnhancer(): void {
    const parent = this.propertiesRef?.nativeElement;
    if (!parent) return;

    if (this.constraintObserver) {
      this.constraintObserver.disconnect();
      this.constraintObserver = undefined;
    }

    const SUPPORTED_CONSTRAINTS = [
      { value: 'required', label: 'required (Required)' },
      { value: 'minlength', label: 'minlength (Minimum Length)' },
      { value: 'maxlength', label: 'maxlength (Maximum Length)' },
      { value: 'min', label: 'min (Minimum Value)' },
      { value: 'max', label: 'max (Maximum Value)' },
      { value: 'pattern', label: 'pattern (Regex / Regular Expression)' },
      { value: 'readonly', label: 'readonly (Read Only)' }
    ];

    const CONFIG_PLACEHOLDERS: { [key: string]: string } = {
      required: 'true',
      readonly: 'true',
      minlength: 'e.g. 5 (minimum characters)',
      maxlength: 'e.g. 50 (maximum characters)',
      min: 'e.g. 100 (minimum numeric value)',
      max: 'e.g. 5000 (maximum numeric value)',
      pattern: 'e.g. ^[A-Z]{3}-\\d{3}$ (regex)'
    };

    const UNWANTED_GROUPS = [
      'history clean up',
      'history time to live',
      'task list',
      'tasklist',
      'candidate starter',
      'external task',
      'job execution',
      'job configuration',
      'execution listeners',
      'extension properties',
      'executable',
      'isexecutable',
      'start indicator',
      'initiator',
      'asynchronous continuation',
      'asynchronous continuations',
      'asynchronous',
      'async',
      'async before',
      'async after',
      'script format',
      'script body',
      'script language',
      'condition script'
    ];

    const UNWANTED_IDS = [
      'historytimetolive',
      'historycleanup',
      'tasklist',
      'candidatestarter',
      'externaltask',
      'jobexecution',
      'jobconfiguration',
      'executionlisteners',
      'extensionproperties',
      'isexecutable',
      'executable',
      'startindicator',
      'initiator',
      'asynchronouscontinuation',
      'asynchronouscontinuations',
      'asynchronous',
      'async',
      'asyncbefore',
      'asyncafter',
      'scriptformat',
      'scriptbody',
      'scriptlanguage',
      'conditionscript'
    ];

    const hideUnsupportedGroups = () => {
      const groupEls = parent.querySelectorAll('.bio-properties-panel-group, [data-group-id]');
      groupEls.forEach((groupEl) => {
        const groupId = (groupEl.getAttribute('data-group-id') || '').toLowerCase().replace(/[^a-z0-9]/g, '');
        const headerEl = groupEl.querySelector('.bio-properties-panel-group-header-title, .bio-properties-panel-group-title, h3, h4, .group-title, .bio-properties-panel-group-header');
        const headerText = (headerEl?.textContent || '').toLowerCase().trim();

        const isUnwanted = UNWANTED_IDS.some(id => groupId.includes(id)) ||
                           UNWANTED_GROUPS.some(name => headerText.includes(name));

        if (isUnwanted) {
          (groupEl as HTMLElement).style.display = 'none';
        }
      });

      const entryEls = parent.querySelectorAll('.bio-properties-panel-entry, [data-entry-id]');
      entryEls.forEach((entryEl) => {
        const entryId = (entryEl.getAttribute('data-entry-id') || '').toLowerCase().replace(/[^a-z0-9]/g, '');
        const labelEl = entryEl.querySelector('label, .bio-properties-panel-label');
        const labelText = (labelEl?.textContent || '').toLowerCase().trim();

        const isUnwantedEntry = UNWANTED_IDS.some(id => entryId.includes(id)) ||
                                UNWANTED_GROUPS.some(name => labelText.includes(name));

        if (isUnwantedEntry) {
          (entryEl as HTMLElement).style.display = 'none';
        }
      });

      const conditionSelects = parent.querySelectorAll('[data-entry-id*="condition"] select, select[name*="condition"], [data-entry-id*="conditionType"] select');
      conditionSelects.forEach((selectEl) => {
        const select = selectEl as HTMLSelectElement;
        Array.from(select.options).forEach((opt) => {
          const val = (opt.value || '').toLowerCase();
          const txt = (opt.textContent || '').toLowerCase();
          if (val === 'script' || txt.includes('script')) {
            opt.remove();
          }
        });
      });
    };

    const enhanceConstraintEntries = () => {
      hideUnsupportedGroups();

      const nameEntries = parent.querySelectorAll('[data-entry-id*="-constraint-"][data-entry-id$="-name"]');
      nameEntries.forEach((entryEl) => {
        const input = entryEl.querySelector('input') as HTMLInputElement;
        if (!input || input.dataset['enhanced']) return;

        input.dataset['enhanced'] = 'true';
        input.style.display = 'none';

        const select = document.createElement('select');
        select.className = 'bio-properties-panel-input bio-properties-panel-select';
        select.style.width = '100%';
        select.style.backgroundColor = '#181b22';
        select.style.color = '#ffffff';
        select.style.border = '1px solid rgba(255, 255, 255, 0.2)';
        select.style.borderRadius = '4px';
        select.style.padding = '6px 8px';
        select.style.marginTop = '4px';

        const placeholderOpt = document.createElement('option');
        placeholderOpt.value = '';
        placeholderOpt.textContent = '-- Select Validation Rule --';
        select.appendChild(placeholderOpt);

        let isKnown = false;
        SUPPORTED_CONSTRAINTS.forEach((c) => {
          const opt = document.createElement('option');
          opt.value = c.value;
          opt.textContent = c.label;
          if (input.value && input.value.toLowerCase() === c.value) {
            opt.selected = true;
            isKnown = true;
          }
          select.appendChild(opt);
        });

        if (input.value && !isKnown) {
          const customOpt = document.createElement('option');
          customOpt.value = input.value;
          customOpt.textContent = input.value + ' (Custom)';
          customOpt.selected = true;
          select.appendChild(customOpt);
        }

        input.parentNode?.insertBefore(select, input.nextSibling);

        const entryId = entryEl.getAttribute('data-entry-id') || '';
        const configEntryId = entryId.replace(/-name$/, '-config');
        const configEntryEl = parent.querySelector(`[data-entry-id="${configEntryId}"]`);
        const configInput = configEntryEl?.querySelector('input') as HTMLInputElement;

        const updateConfigPlaceholder = (selectedVal: string) => {
          if (configInput) {
            configInput.placeholder = CONFIG_PLACEHOLDERS[selectedVal.toLowerCase()] || 'e.g. configuration value';
          }
        };

        if (input.value) {
          updateConfigPlaceholder(input.value);
        }

        select.addEventListener('change', () => {
          const chosen = select.value;
          const nativeSetter = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, 'value')?.set;
          if (nativeSetter) {
            nativeSetter.call(input, chosen);
          } else {
            input.value = chosen;
          }
          input.dispatchEvent(new Event('input', { bubbles: true }));
          input.dispatchEvent(new Event('change', { bubbles: true }));
          updateConfigPlaceholder(chosen);
        });
      });
    };

    this.constraintObserver = new MutationObserver(() => {
      enhanceConstraintEntries();
    });

    this.constraintObserver.observe(parent, { childList: true, subtree: true });
    setTimeout(enhanceConstraintEntries, 100);
  }
}
