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
  imports: [CommonModule],
  encapsulation: ViewEncapsulation.None
})
export class BpmnDesignerComponent implements AfterViewInit, OnChanges, OnDestroy {
  @ViewChild('canvasRef', { static: true }) private canvasRef!: ElementRef<HTMLDivElement>;
  @ViewChild('propertiesRef', { static: true }) private propertiesRef!: ElementRef<HTMLDivElement>;

  @Input() payload: string = '';
  @Input() readonly: boolean = false;
  @Output() xmlChange = new EventEmitter<string>();

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
          ElementTemplateChooserModule
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
            if (!elementType.includes('gateway')) {
              delete entries['replace'];
            }
            return entries;
          };
        }
      } catch (e) {
        console.warn('Context pad entries override notice:', e);
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

    // Listen for diagram changes to emit xmlChange if not readonly
    if (!this.readonly) {
      const eventBus = this.bpmnModeler.get('eventBus');
      eventBus.on('commandStack.changed', () => {
        this.emitCurrentXml();
      });
    }

    const initialXml = this.payload && this.payload.trim().startsWith('<?xml')
      ? this.payload
      : DEFAULT_BPMN_XML;

    this.importXml(initialXml);
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
      'async after'
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
      'asyncafter'
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
