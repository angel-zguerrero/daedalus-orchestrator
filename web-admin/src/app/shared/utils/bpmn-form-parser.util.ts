export interface FormFieldValue {
  id: string;
  name: string;
}

export interface FormFieldConstraint {
  name: string;
  config?: string;
}

export interface GeneratedFormField {
  id: string;
  label: string;
  type: string; // 'string' | 'long' | 'integer' | 'boolean' | 'date' | 'enum'
  defaultValue?: string;
  values?: FormFieldValue[];
  required?: boolean;
  minlength?: number;
  maxlength?: number;
  min?: number;
  max?: number;
  readonly?: boolean;
  pattern?: string;
  constraints?: FormFieldConstraint[];
}

export class BpmnFormParserUtil {
  /**
   * Parses BPMN XML string and extracts Generated Task Form fields (<camunda:formData>)
   * from the StartEvent node along with rich validation constraints.
   */
  public static extractStartFormFields(xmlString: string): GeneratedFormField[] {
    if (!xmlString || typeof xmlString !== 'string' || !xmlString.trim()) {
      return [];
    }

    let rawXml = xmlString.trim();

    // If payload is base64 encoded string, attempt decoding
    if (!rawXml.startsWith('<')) {
      try {
        const decoded = atob(rawXml);
        if (decoded.trim().startsWith('<')) {
          rawXml = decoded.trim();
        }
      } catch {
        // Not base64
      }
    }

    // Handle JSON stringified XML (e.g. "\"<?xml...\"")
    if (rawXml.startsWith('"') && rawXml.endsWith('"')) {
      try {
        const unescaped = JSON.parse(rawXml);
        if (typeof unescaped === 'string' && unescaped.trim().startsWith('<')) {
          rawXml = unescaped.trim();
        }
      } catch {
        // ignore
      }
    }

    try {
      const parser = new DOMParser();
      const doc = parser.parseFromString(rawXml, 'text/xml');

      // Check for parsing errors
      const parserError = doc.querySelector('parsererror');
      if (parserError) {
        console.warn('BpmnFormParserUtil: XML parse error', parserError.textContent);
        return [];
      }

      // Find startEvent nodes
      const allElements = doc.getElementsByTagName('*');
      let startNode: Element | null = null;

      for (let i = 0; i < allElements.length; i++) {
        const el = allElements[i];
        if (el.localName === 'startEvent') {
          startNode = el;
          break;
        }
      }

      if (!startNode) {
        return [];
      }

      // Find formData inside startNode
      let formDataNode: Element | null = null;
      const children = startNode.getElementsByTagName('*');
      for (let i = 0; i < children.length; i++) {
        if (children[i].localName === 'formData') {
          formDataNode = children[i];
          break;
        }
      }

      if (!formDataNode) {
        return [];
      }

      const formFields: GeneratedFormField[] = [];
      const fieldChildren = formDataNode.getElementsByTagName('*');

      for (let i = 0; i < fieldChildren.length; i++) {
        const fieldEl = fieldChildren[i];
        if (fieldEl.localName !== 'formField') {
          continue;
        }

        const id = fieldEl.getAttribute('id') || '';
        if (!id) continue;

        const label = fieldEl.getAttribute('label') || id;
        const type = (fieldEl.getAttribute('type') || 'string').toLowerCase();
        const defaultValue = fieldEl.getAttribute('defaultValue') || '';

        // Extract enum values and constraints
        const values: FormFieldValue[] = [];
        const constraintsList: FormFieldConstraint[] = [];
        let isRequired = false;
        let minlength: number | undefined;
        let maxlength: number | undefined;
        let minVal: number | undefined;
        let maxVal: number | undefined;
        let isReadonly = false;
        let patternStr: string | undefined;

        const subChildren = fieldEl.getElementsByTagName('*');
        for (let j = 0; j < subChildren.length; j++) {
          const sub = subChildren[j];
          if (sub.localName === 'value') {
            const valId = sub.getAttribute('id') || '';
            const valName = sub.getAttribute('name') || valId;
            if (valId) {
              values.push({ id: valId, name: valName });
            }
          } else if (sub.localName === 'constraint') {
            const cName = (sub.getAttribute('name') || '').toLowerCase();
            const cConfig = sub.getAttribute('config') || sub.getAttribute('value') || '';
            constraintsList.push({ name: cName, config: cConfig });

            if (cName === 'required' && (cConfig === 'true' || cConfig === '' || cConfig === null)) {
              isRequired = true;
            } else if (cName === 'readonly' && (cConfig === 'true' || cConfig === '' || cConfig === null)) {
              isReadonly = true;
            } else if ((cName === 'minlength' || cName === 'min_length' || cName === 'min') && cConfig && !isNaN(Number(cConfig))) {
              const num = Number(cConfig);
              if (type === 'long' || type === 'integer' || type === 'number') {
                minVal = num;
                minlength = num;
              } else {
                minlength = num;
                minVal = num;
              }
            } else if ((cName === 'maxlength' || cName === 'max_length' || cName === 'max') && cConfig && !isNaN(Number(cConfig))) {
              const num = Number(cConfig);
              if (type === 'long' || type === 'integer' || type === 'number') {
                maxVal = num;
                maxlength = num;
              } else {
                maxlength = num;
                maxVal = num;
              }
            } else if (cName === 'pattern' && cConfig) {
              patternStr = cConfig;
            }
          }
        }

        formFields.push({
          id,
          label,
          type,
          defaultValue,
          values: values.length > 0 ? values : undefined,
          required: isRequired,
          minlength,
          maxlength,
          min: minVal,
          max: maxVal,
          readonly: isReadonly,
          pattern: patternStr,
          constraints: constraintsList.length > 0 ? constraintsList : undefined
        });
      }

      return formFields;
    } catch (err) {
      console.error('BpmnFormParserUtil error parsing BPMN form fields:', err);
      return [];
    }
  }
}
