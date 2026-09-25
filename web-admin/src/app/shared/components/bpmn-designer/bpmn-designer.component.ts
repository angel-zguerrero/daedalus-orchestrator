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
import { ActivityTemplatesService, ActivityTemplate } from '../../../views/activity-templates/services/activity-templates.service';

export interface FrozenPropertyInfo {
  label: string;
  value: any;
  bindingName: string;
}

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
  @ViewChild('fileInputRef') private fileInputRef?: ElementRef<HTMLInputElement>;
  @ViewChild('errorsSectionRef') private errorsSectionRef?: ElementRef<HTMLDivElement>;

  @Input() payload: string = '';
  @Input() readonly: boolean = false;
  @Input() scope: 'global' | 'tenant' = 'global';
  @Input() tenantCode: string = '';
  @Output() xmlChange = new EventEmitter<string>();
  @Output() designErrorsChange = new EventEmitter<{ hasErrors: boolean; errors: string[] }>();

  public currentLintErrors: string[] = [];
  public hasLintErrors: boolean = false;
  public selectedElementInfo: any = null;

  public scrollToErrorsSection(): void {
    if (this.errorsSectionRef && this.errorsSectionRef.nativeElement) {
      this.errorsSectionRef.nativeElement.scrollIntoView({ behavior: 'smooth', block: 'start' });
    }
  }

  public inspectElement(element: any): void {
    if (!element) {
      this.selectedElementInfo = null;
      return;
    }
    const bo = element.businessObject || {};
    const attrs: Array<{ key: string; value: string; category?: string }> = [];

    const rawType = (element.type || bo.$type || '').replace(/^bpmn:/, '');
    let cleanType = rawType;
    switch (rawType) {
      case 'ServiceTask': cleanType = 'Service Task'; break;
      case 'UserTask': cleanType = 'User Task'; break;
      case 'SendTask': cleanType = 'Send Task'; break;
      case 'ReceiveTask': cleanType = 'Receive Task'; break;
      case 'ScriptTask': cleanType = 'Script Task'; break;
      case 'BusinessRuleTask': cleanType = 'Business Rule Task'; break;
      case 'ExclusiveGateway': cleanType = 'Exclusive Gateway (XOR)'; break;
      case 'ParallelGateway': cleanType = 'Parallel Gateway (AND)'; break;
      case 'InclusiveGateway': cleanType = 'Inclusive Gateway (OR)'; break;
      case 'ComplexGateway': cleanType = 'Complex Gateway'; break;
      case 'EventBasedGateway': cleanType = 'Event-Based Gateway'; break;
      case 'StartEvent': cleanType = 'Start Event'; break;
      case 'EndEvent': cleanType = 'End Event'; break;
      case 'IntermediateCatchEvent': cleanType = 'Intermediate Catch Event'; break;
      case 'IntermediateThrowEvent': cleanType = 'Intermediate Throw Event'; break;
      case 'SequenceFlow': cleanType = 'Sequence Flow'; break;
    }

    const docText = bo.documentation && bo.documentation.length > 0
      ? (typeof bo.documentation[0] === 'string' ? bo.documentation[0] : (bo.documentation[0].text || ''))
      : '';

    const info: any = {
      id: element.id || bo.id || '',
      name: bo.name || (element.id && !element.id.startsWith('Process_') ? element.id : ''),
      type: cleanType,
      rawType: rawType,
      documentation: docText,
      attributes: []
    };

    // Core Properties
    if (bo.topic || (bo.$attrs && bo.$attrs['camunda:topic'])) {
      attrs.push({ key: 'Topic / Queue Name', value: bo.topic || bo.$attrs['camunda:topic'], category: 'Execution' });
    }
    if (bo.type || (bo.$attrs && bo.$attrs['camunda:type'])) {
      attrs.push({ key: 'Job Type', value: bo.type || bo.$attrs['camunda:type'], category: 'Execution' });
    }
    if (bo.assignee || (bo.$attrs && bo.$attrs['camunda:assignee'])) {
      attrs.push({ key: 'Assignee', value: bo.assignee || bo.$attrs['camunda:assignee'], category: 'Assignment' });
    }
    if (bo.candidateGroups || (bo.$attrs && bo.$attrs['camunda:candidateGroups'])) {
      attrs.push({ key: 'Candidate Groups', value: bo.candidateGroups || bo.$attrs['camunda:candidateGroups'], category: 'Assignment' });
    }
    if (bo.candidateUsers || (bo.$attrs && bo.$attrs['camunda:candidateUsers'])) {
      attrs.push({ key: 'Candidate Users', value: bo.candidateUsers || bo.$attrs['camunda:candidateUsers'], category: 'Assignment' });
    }
    if (bo.formKey || (bo.$attrs && bo.$attrs['camunda:formKey'])) {
      attrs.push({ key: 'Form Key', value: bo.formKey || bo.$attrs['camunda:formKey'], category: 'Form' });
    }

    // Sequence Flow details
    if (bo.sourceRef) {
      attrs.push({ key: 'Source Element', value: `${bo.sourceRef.name || bo.sourceRef.id} (${(bo.sourceRef.$type || '').replace(/^bpmn:/, '')})`, category: 'Flow' });
    }
    if (bo.targetRef) {
      attrs.push({ key: 'Target Element', value: `${bo.targetRef.name || bo.targetRef.id} (${(bo.targetRef.$type || '').replace(/^bpmn:/, '')})`, category: 'Flow' });
    }
    if (bo.conditionExpression && bo.conditionExpression.body) {
      attrs.push({ key: 'Condition Expression', value: bo.conditionExpression.body, category: 'Flow' });
    }

    // $attrs loop for extra custom definitions
    if (bo.$attrs) {
      for (const [k, v] of Object.entries(bo.$attrs)) {
        if (typeof v === 'string' && v.trim() && !k.startsWith('xmlns') && !k.startsWith('xsi:')) {
          const cleanKey = k.replace(/^camunda:/, '').replace(/^custom:/, '');
          if (!attrs.some(a => a.key.toLowerCase() === cleanKey.toLowerCase())) {
            attrs.push({ key: cleanKey, value: v, category: 'Configuration' });
          }
        }
      }
    }

    // Extension Elements
    if (bo.extensionElements && bo.extensionElements.values) {
      for (const ext of bo.extensionElements.values) {
        const extType = (ext.$type || '').toLowerCase();

        // Form Data & Form Fields
        if (extType.includes('formdata') && ext.fields) {
          for (const f of ext.fields) {
            let desc = `Type: ${f.type || 'string'}`;
            if (f.label) desc += ` | Label: "${f.label}"`;
            if (f.defaultValue) desc += ` | Default: "${f.defaultValue}"`;

            const constraints: string[] = [];
            if (f.validation && f.validation.constraints) {
              for (const c of f.validation.constraints) {
                constraints.push(`${c.name}=${c.config}`);
              }
            }
            if (constraints.length > 0) {
              desc += ` | Rules: ${constraints.join(', ')}`;
            }

            attrs.push({ key: `Form Field: ${f.id}`, value: desc, category: 'Form Fields' });
          }
        }

        // Input & Output Parameters
        if (extType.includes('inputoutput')) {
          if (ext.inputParameters) {
            for (const inp of ext.inputParameters) {
              const valStr = typeof inp.value === 'string' ? inp.value : JSON.stringify(inp.value);
              attrs.push({ key: `Input: ${inp.name}`, value: valStr, category: 'Inputs & Outputs' });
            }
          }
          if (ext.outputParameters) {
            for (const out of ext.outputParameters) {
              const valStr = typeof out.value === 'string' ? out.value : JSON.stringify(out.value);
              attrs.push({ key: `Output: ${out.name}`, value: valStr, category: 'Inputs & Outputs' });
            }
          }
        }

        // Extension Properties
        if (extType.includes('properties') && ext.values) {
          for (const prop of ext.values) {
            attrs.push({ key: `Property: ${prop.name}`, value: prop.value || '', category: 'Custom Properties' });
          }
        }
      }
    }

    info.attributes = attrs;
    this.selectedElementInfo = info;
  }

  private bpmnModeler!: any;
  private isInitialized = false;
  private lastEmittedXml: string = '';
  private constraintObserver?: MutationObserver;

  private customElementTemplates: any[] = [];
  private customTemplatesCursor: string = '';
  private loadingCustomTemplates: boolean = false;
  private frozenPropertyMapByTemplate = new Map<string, Map<string, FrozenPropertyInfo>>();

  constructor(private activityTemplatesService: ActivityTemplatesService) {}

  ngAfterViewInit(): void {
    this.initModeler();
  }

  ngOnChanges(changes: SimpleChanges): void {
    if (changes['readonly'] && this.isInitialized && !changes['readonly'].firstChange) {
      this.reinitModeler();
    } else if ((changes['tenantCode'] || changes['scope']) && this.isInitialized && !this.readonly) {
      this.customElementTemplates = [];
      this.customTemplatesCursor = '';
      this.loadCustomActivityTemplates(false);
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
        (elementTemplatesData as any[]).forEach((tpl: any) => {
          this.registerFrozenProperties([tpl.id, tpl.name], tpl);
        });

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

      this.customElementTemplates = [];
      this.customTemplatesCursor = '';
      this.loadCustomActivityTemplates(false);

      this.setupPropertiesPanelConstraintEnhancer();
    }

    this.isInitialized = true;

    // Listen for diagram events
    if (this.bpmnModeler) {
      const eventBus = this.bpmnModeler.get('eventBus');
      if (eventBus) {
        if (!this.readonly) {
          eventBus.on('commandStack.changed', () => {
            this.emitCurrentXml();
            this.runLintValidation();
          });
          eventBus.on('import.done', () => {
            setTimeout(() => this.runLintValidation(), 150);
          });
        }
        eventBus.on('element.click', (event: any) => {
          const element = event.element;
          if (element && element.id && !element.id.startsWith('Process_') && !element.id.includes('_plane')) {
            this.inspectElement(element);
          } else {
            this.selectedElementInfo = null;
          }
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
      this.updateCanvasOverlays(new Map());
      return { hasErrors: false, errors: [] };
    }

    const errors: string[] = [];
    const elementErrorMap = new Map<string, string[]>();

    const addElementError = (elementId: string, msg: string) => {
      errors.push(msg);
      if (elementId) {
        const existing = elementErrorMap.get(elementId) || [];
        existing.push(msg);
        elementErrorMap.set(elementId, existing);
      }
    };

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
              addElementError(id, `Sequence Flow ${displayName} is disconnected (missing source or target).`);
            }
            return; // Connections are edges, not nodes; skip shape incoming/outgoing checks!
          }

          // Shape / Node checks
          const incoming = el.incoming || [];
          const outgoing = el.outgoing || [];

          if (type === 'StartEvent') {
            startEventCount++;
            if (outgoing.length === 0) {
              addElementError(id, `Start Event ${displayName} has no outgoing connection.`);
            }
          } else if (type === 'EndEvent') {
            endEventCount++;
            if (incoming.length === 0) {
              addElementError(id, `End Event ${displayName} has no incoming connection.`);
            }
          } else if (type === 'BoundaryEvent') {
            if (outgoing.length === 0) {
              addElementError(id, `Boundary Event ${displayName} has no outgoing connection.`);
            }
          } else {
            // Regular flow nodes (Tasks, Gateways, SubProcesses, etc.)
            if (incoming.length === 0 && outgoing.length === 0) {
              addElementError(id, `Element ${displayName} (${type}) is disconnected (has no connections).`);
            } else if (incoming.length === 0) {
              addElementError(id, `Element ${displayName} (${type}) has no incoming connection.`);
            } else if (outgoing.length === 0) {
              addElementError(id, `Element ${displayName} (${type}) has no outgoing connection.`);
            }
          }

          // 1. Implicit Split Rule (no-implicit-split):
          // Non-gateway elements (StartEvent, Task, UserTask, etc.) MUST NOT have multiple outgoing sequence flows.
          if (!type.includes('Gateway') && outgoing.length > 1) {
            addElementError(
              id,
              `Implicit Split Error: ${displayName} (${type}) has multiple outgoing sequence flows without a Gateway. Use an explicit Gateway (e.g. Parallel or Exclusive Gateway) to split execution paths.`
            );
          }

          // 2. Gateway & Split Divergence checks:
          if (outgoing.length > 1) {
            const divType = type.includes('Gateway') ? type : 'ImplicitParallelSplit';
            const defaultFlowId = el.businessObject?.default?.id;

            // Condition check on outgoing flows (ONLY for conditional divergent gateways: Exclusive, Inclusive, etc.)
            const isConditionalGateway = type.includes('Gateway') && divType !== 'ParallelGateway';
            if (isConditionalGateway) {
              outgoing.forEach((flow: any) => {
                const cond = (flow.businessObject?.conditionExpression?.body || '').trim();
                const isDefault = defaultFlowId && flow.id === defaultFlowId;
                if (!cond && !isDefault) {
                  const targetId = flow.target?.id || 'target';
                  const targetRawName = (flow.target?.businessObject?.name || '').trim();
                  const targetDisplayName = targetRawName ? `'${targetRawName}' (${targetId})` : `'${targetId}'`;
                  addElementError(id, `Gateway ${displayName} flow to ${targetDisplayName} has no condition and is not marked as default flow.`);
                }
              });
            }

            // Nesting-aware Gateway Symmetry, Deadlock & Token Multiplication Check
            interface QueueItem {
              node: any;
              stack: { id: string; type: string; displayName: string }[];
            }
            const queue: QueueItem[] = [];
            const visitedState = new Set<string>();

            outgoing.forEach((flow: any) => {
              if (flow && flow.target) {
                queue.push({
                  node: flow.target,
                  stack: [{ id, type: divType, displayName }]
                });
              }
            });

            while (queue.length > 0) {
              const item = queue.shift();
              if (!item) continue;
              const { node, stack } = item;
              if (!node || !node.id) continue;

              const stackKey = stack.map(s => s.id).join(',');
              const stateKey = `${node.id}:${stackKey}`;
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
                  const topDiv = currentStack[currentStack.length - 1];
                  const topDivType = topDiv.type;
                  const convType = nodeType;
                  const convId = node.id;

                  const pairKey = `${topDiv.id}->${convId}`;
                  if (!reportedGatewayPairs.has(pairKey)) {
                    const convRawName = (node.businessObject?.name || '').trim();
                    const convDisplayName = convRawName ? `'${convRawName}' (${convId})` : `'${convId}'`;

                    // Token Multiplication Error: Implicit Parallel Split or Parallel Gateway converging into Exclusive Gateway (XOR)
                    if ((topDivType === 'ImplicitParallelSplit' || topDivType === 'ParallelGateway') && convType === 'ExclusiveGateway') {
                      reportedGatewayPairs.add(pairKey);
                      const divName = topDivType === 'ImplicitParallelSplit' ? `Implicit parallel split from ${topDiv.displayName}` : `Parallel Gateway (AND) ${topDiv.displayName}`;
                      const msg = `Token Multiplication Error: ${divName} converges into Exclusive Gateway (XOR) ${convDisplayName}. An Exclusive Gateway does not synchronize parallel tokens, causing downstream tasks to execute multiple times.`;
                      addElementError(topDiv.id, msg);
                      addElementError(convId, msg);
                    } else if (topDivType === 'ExclusiveGateway' && convType === 'ParallelGateway') {
                      reportedGatewayPairs.add(pairKey);
                      const divTypeName = getGatewayTypeName(topDivType);
                      const convTypeName = getGatewayTypeName(convType);
                      const msg = `Deadlock Error: ${divTypeName} ${topDiv.displayName} diverges into ${convTypeName} ${convDisplayName}. A Parallel Gateway waits for all incoming branches, causing a permanent deadlock because an Exclusive Gateway only activates one branch.`;
                      addElementError(topDiv.id, msg);
                      addElementError(convId, msg);
                    } else if (topDivType !== 'ImplicitParallelSplit' && topDivType !== convType) {
                      reportedGatewayPairs.add(pairKey);
                      const divTypeName = getGatewayTypeName(topDivType);
                      const convTypeName = getGatewayTypeName(convType);
                      const msg = `Gateway Asymmetry Error: ${divTypeName} ${topDiv.displayName} diverges into ${convTypeName} ${convDisplayName}. Gateways must be symmetric (divergence and convergence must use the same gateway type).`;
                      addElementError(topDiv.id, msg);
                      addElementError(convId, msg);
                    }
                  }

                  currentStack.pop();
                }
              }

              // If node is another divergent element (Gateway or Implicit Split with outgoing > 1), push to stack
              if (nodeOutgoingCount > 1 && node.id !== id) {
                const nestedDivType = nodeType.includes('Gateway') ? nodeType : 'ImplicitParallelSplit';
                const nodeRawName = (node.businessObject?.name || '').trim();
                const nodeDisplayName = nodeRawName ? `'${nodeRawName}' (${node.id})` : `'${node.id}'`;
                currentStack.push({ id: node.id, type: nestedDivType, displayName: nodeDisplayName });
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
    this.updateCanvasOverlays(elementErrorMap);
    const result = { hasErrors: this.hasLintErrors, errors: this.currentLintErrors };
    this.designErrorsChange.emit(result);
    return result;
  }

  private updateCanvasOverlays(elementErrorMap: Map<string, string[]>): void {
    if (!this.bpmnModeler) return;
    try {
      const overlays = this.bpmnModeler.get('overlays');
      const canvas = this.bpmnModeler.get('canvas');
      const elementRegistry = this.bpmnModeler.get('elementRegistry');

      if (!overlays || !canvas || !elementRegistry) return;

      // 1. Clear previous lint overlays and markers
      overlays.remove({ type: 'lint-badge-overlay' });
      elementRegistry.getAll().forEach((el: any) => {
        if (el && el.id) {
          canvas.removeMarker(el.id, 'highlight-lint-error');
        }
      });

      // 2. Add visual markers & overlays for elements with errors
      elementErrorMap.forEach((messages, elementId) => {
        const element = elementRegistry.get(elementId);
        if (!element) return;

        // Add red glowing border marker
        canvas.addMarker(elementId, 'highlight-lint-error');

        // Create HTML overlay badge
        const badgeEl = document.createElement('div');
        badgeEl.className = 'bpmn-lint-badge-overlay';
        badgeEl.setAttribute('data-tooltip', messages.join('\n\n'));
        badgeEl.innerHTML = `<span class="badge-icon">!</span><span class="badge-count">${messages.length}</span>`;

        overlays.add(elementId, 'lint-badge-overlay', {
          position: {
            top: -10,
            right: -10
          },
          html: badgeEl
        });
      });
    } catch (e) {
      console.warn('Canvas overlay update notice:', e);
    }
  }

  public refresh(): void {
    if (!this.bpmnModeler) return;
    if (!this.readonly) {
      this.loadCustomActivityTemplates(false);
    }
    try {
      const canvas = this.bpmnModeler.get('canvas');
      canvas.resized();
      canvas.zoom('fit-viewport');
    } catch {
      // ignore resize error
    }
  }

  public highlightElements(markers: Array<{ id: string; type: 'active' | 'error' }>): void {
    if (!this.bpmnModeler) return;
    try {
      const canvas = this.bpmnModeler.get('canvas');
      if (!canvas) return;
      markers.forEach(m => {
        const cssClass = m.type === 'active' ? 'highlight-active' : 'highlight-error';
        canvas.addMarker(m.id, cssClass);
      });
    } catch (e) {
      console.warn('Failed to add markers:', e);
    }
  }

  public clearHighlights(): void {
    if (!this.bpmnModeler) return;
    try {
      const canvas = this.bpmnModeler.get('canvas');
      const elementRegistry = this.bpmnModeler.get('elementRegistry');
      if (!canvas || !elementRegistry) return;
      elementRegistry.getAll().forEach((el: any) => {
        if (el && el.id) {
          canvas.removeMarker(el.id, 'highlight-active');
          canvas.removeMarker(el.id, 'highlight-error');
        }
      });
    } catch (e) {
      console.warn('Failed to clear markers:', e);
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

  public triggerFileUpload(): void {
    if (this.fileInputRef?.nativeElement) {
      this.fileInputRef.nativeElement.value = '';
      this.fileInputRef.nativeElement.click();
    }
  }

  public async onFileSelected(event: Event): Promise<void> {
    const input = event.target as HTMLInputElement;
    if (!input.files || input.files.length === 0) return;

    const file = input.files[0];
    try {
      const content = await file.text();
      if (content && content.trim()) {
        await this.importXml(content);
        await this.emitCurrentXml();
        this.runLintValidation();
      }
    } catch (err) {
      console.error('Failed to read BPMN XML file:', err);
    }
  }

  public async downloadXml(): Promise<void> {
    const xml = await this.getXml();
    if (!xml || !xml.trim()) return;

    const blob = new Blob([xml], { type: 'application/xml;charset=utf-8;' });
    const link = document.createElement('a');
    const url = URL.createObjectURL(blob);
    link.setAttribute('href', url);
    link.setAttribute('download', 'workflow-diagram.bpmn');
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
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

      // Enforce read-only state on any burned/frozen properties of custom Activity Templates
      const allPropInputs = parent.querySelectorAll(
        '[data-entry-id*="custom-entry-"] input, [data-entry-id*="custom-entry-"] textarea, [data-entry-id*="custom-entry-"] select'
      );
      allPropInputs.forEach((el) => {
        const htmlEl = el as HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement;
        if (htmlEl.disabled) {
          htmlEl.style.opacity = '0.65';
          htmlEl.style.cursor = 'not-allowed';
          htmlEl.title = 'Burned / Locked by Activity Template';
        }
      });

      // Also check by label & property against frozenPropertyMapByTemplate for the currently selected element
      try {
        const selection = this.bpmnModeler?.get('selection')?.get?.() || [];
        const selectedEl = selection[0];
        const tplId = (
          selectedEl?.businessObject?.modelerTemplate ||
          selectedEl?.businessObject?.$attrs?.['camunda:modelerTemplate'] ||
          ''
        ).trim().toLowerCase();

        let frozenMap = tplId ? this.frozenPropertyMapByTemplate.get(tplId) : undefined;
        if (!frozenMap && tplId) {
          for (const [k, mapVal] of this.frozenPropertyMapByTemplate.entries()) {
            if (k === tplId || k.includes(tplId) || tplId.includes(k)) {
              frozenMap = mapVal;
              break;
            }
          }
        }

        if (frozenMap && frozenMap.size > 0) {
          const entries = parent.querySelectorAll('.bio-properties-panel-entry, [data-entry-id]');
          entries.forEach((entryEl) => {
            const lbl = entryEl.querySelector('label, .bio-properties-panel-label');
            const lblTxt = (lbl?.textContent || '').trim().toLowerCase();
            const entryId = (entryEl.getAttribute('data-entry-id') || '').toLowerCase();

            frozenMap!.forEach((frozenInfo, labelKey) => {
              const bName = (frozenInfo.bindingName || '').toLowerCase().replace(/^camunda:/, '');
              const isMatchLabel = lblTxt && (lblTxt === labelKey || lblTxt === frozenInfo.label.toLowerCase());
              const isMatchBinding = entryId && bName && entryId.includes(bName);
              const isResultVarMatch = (bName.includes('resultvariable') || labelKey.includes('result variable')) &&
                                      (entryId.includes('resultvariable') || lblTxt.includes('result variable'));

              if (isMatchLabel || isMatchBinding || isResultVarMatch) {
                const inputs = entryEl.querySelectorAll('input, textarea, select');
                inputs.forEach((inp: any) => {
                  if (frozenInfo.value !== undefined && frozenInfo.value !== null && frozenInfo.value !== '') {
                    const strVal = String(frozenInfo.value);
                    if (inp.type === 'checkbox') {
                      const shouldCheck = strVal.toLowerCase() === 'true';
                      if (inp.checked !== shouldCheck) {
                        inp.checked = shouldCheck;
                        inp.dispatchEvent(new Event('input', { bubbles: true }));
                        inp.dispatchEvent(new Event('change', { bubbles: true }));
                      }
                    } else {
                      if (!inp.value || inp.value !== strVal) {
                        inp.value = strVal;
                        inp.dispatchEvent(new Event('input', { bubbles: true }));
                        inp.dispatchEvent(new Event('change', { bubbles: true }));
                      }
                    }
                  }
                  inp.disabled = true;
                  inp.readOnly = true;
                  inp.style.opacity = '0.65';
                  inp.style.cursor = 'not-allowed';
                  inp.title = 'Burned / Locked by Activity Template';
                });
              }
            });
          });
        }
      } catch {
        // ignore selection inspection error
      }

      // Inject lazy-fetch "Load More Templates" button in properties panel template section if more cursor pages exist
      if (this.customTemplatesCursor && !parent.querySelector('.lazy-load-templates-btn')) {
        const templateGroup = parent.querySelector('[data-group-id*="template"], .bio-properties-panel-template-header');
        if (templateGroup) {
          const btn = document.createElement('button');
          btn.type = 'button';
          btn.className = 'btn btn-sm btn-outline-info w-100 mt-1 lazy-load-templates-btn';
          btn.textContent = 'Load More Activity Templates...';
          btn.onclick = () => this.loadCustomActivityTemplates(true);
          templateGroup.appendChild(btn);
        }
      }

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

  private registerFrozenProperties(keys: (string | undefined)[], obj: any): void {
    if (!obj || !Array.isArray(obj.properties)) return;
    const propMap = new Map<string, FrozenPropertyInfo>();

    obj.properties.forEach((p: any) => {
      const bName = (p.binding?.name || '').trim();
      if (bName === 'resultVariable' || bName === 'camunda:resultVariable') {
        p.binding = {
          type: 'property',
          name: 'camunda:resultVariable'
        };
      }
      if (p.value !== undefined && p.value !== null && typeof p.value !== 'boolean') {
        p.value = String(p.value);
      }
      if (p.editable === false && p.label) {
        const lblTxt = String(p.label).trim();
        const info: FrozenPropertyInfo = {
          label: lblTxt,
          value: p.value !== undefined ? p.value : '',
          bindingName: bName || p.label
        };
        propMap.set(lblTxt.toLowerCase(), info);
        if (bName === 'resultVariable' || bName === 'camunda:resultVariable') {
          propMap.set('result variable', info);
          propMap.set('response result variable name', info);
        }
      }
    });

    if (propMap.size > 0) {
      keys.forEach((k) => {
        if (k && k.trim()) {
          const keyKey = k.trim().toLowerCase();
          const existing = this.frozenPropertyMapByTemplate.get(keyKey) || new Map<string, FrozenPropertyInfo>();
          propMap.forEach((val, keyName) => existing.set(keyName, val));
          this.frozenPropertyMapByTemplate.set(keyKey, existing);
        }
      });
    }
  }

  public loadCustomActivityTemplates(append: boolean = false): void {
    if (this.readonly || this.loadingCustomTemplates || !this.bpmnModeler) {
      return;
    }
    this.loadingCustomTemplates = true;
    const effectiveTenantCode = this.scope === 'tenant' ? this.tenantCode : '';
    const cursorToUse = append ? this.customTemplatesCursor : '';

    this.activityTemplatesService.getForDesigner(effectiveTenantCode, 20, cursorToUse).subscribe({
      next: (res: any) => {
        const list: ActivityTemplate[] = res?.Entities || res?.entities || [];
        this.customTemplatesCursor = res?.Cursor || res?.cursor || '';

        const parsedList: any[] = [];
        list.forEach((item) => {
          const decoded = this.activityTemplatesService.decodePayload(item.payload);
          if (!decoded) return;
          try {
            const obj = JSON.parse(decoded);
            const tplId = item.code || obj.id || item.id;
            obj.id = tplId;
            obj.name = item.name || obj.name;
            obj.description =
              item.description ||
              obj.description ||
              `Activity Family: ${item.activityFamily || 'default'}`;
            obj.category = {
              id: item.activityFamily || 'custom',
              name: `Family: ${item.activityFamily || 'default'} (${item.scope === 'global' ? 'Global' : 'Tenant'})`
            };

            this.registerFrozenProperties(
              [tplId, item.code, item.id, obj.id, item.parentTemplateId, item.rootActivity],
              obj
            );
            parsedList.push(obj);
          } catch {
            // Skip malformed template JSON
          }
        });

        if (append) {
          const existingIds = new Set(this.customElementTemplates.map((t) => t.id));
          parsedList.forEach((p) => {
            if (!existingIds.has(p.id)) {
              this.customElementTemplates.push(p);
            }
          });
        } else {
          this.customElementTemplates = parsedList;
        }

        // Built-in templates ALWAYS come first, followed by custom templates
        const combinedTemplates = [...(elementTemplatesData as any[]), ...this.customElementTemplates];
        try {
          const loader = this.bpmnModeler.get('elementTemplatesLoader');
          if (loader) {
            loader._loadTemplates = combinedTemplates;
            if (loader.setTemplates) {
              loader.setTemplates(combinedTemplates);
            }
          }
          const elementTemplates = this.bpmnModeler.get('elementTemplates');
          if (elementTemplates && elementTemplates.set) {
            elementTemplates.set(combinedTemplates);
          }
        } catch (e) {
          console.warn('Failed to update custom element templates:', e);
        }
        this.loadingCustomTemplates = false;
      },
      error: () => {
        this.loadingCustomTemplates = false;
      }
    });
  }
}
