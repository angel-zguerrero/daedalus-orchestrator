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
}
