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

  ngAfterViewInit(): void {
    this.initModeler();
  }

  ngOnChanges(changes: SimpleChanges): void {
    if (changes['payload'] && this.isInitialized && !changes['payload'].firstChange) {
      this.importXml(this.payload || DEFAULT_BPMN_XML);
    }
  }

  ngOnDestroy(): void {
    if (this.bpmnModeler) {
      this.bpmnModeler.destroy();
    }
  }

  private initModeler(): void {
    this.bpmnModeler = new BpmnModeler({
      container: this.canvasRef.nativeElement,
      propertiesPanel: {
        parent: this.propertiesRef.nativeElement
      },
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
    });

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

    this.isInitialized = true;

    // Listen for diagram changes to emit xmlChange
    const eventBus = this.bpmnModeler.get('eventBus');
    eventBus.on('commandStack.changed', () => {
      this.emitCurrentXml();
    });

    const initialXml = this.payload && this.payload.trim().startsWith('<?xml')
      ? this.payload
      : DEFAULT_BPMN_XML;

    this.importXml(initialXml);
  }

  public async importXml(xml: string): Promise<void> {
    if (!this.bpmnModeler) return;
    try {
      const xmlToLoad = xml && xml.trim().startsWith('<?xml') ? xml : DEFAULT_BPMN_XML;
      await this.bpmnModeler.importXML(xmlToLoad);
      setTimeout(() => {
        try {
          const canvas = this.bpmnModeler.get('canvas');
          canvas.resized();
          canvas.zoom('fit-viewport');
        } catch {
          // ignore canvas resize error if destroyed
        }
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
}
