const fs = require('fs');

const fileName = 'queues.csv';
const totalRecords = 30;

const namespaces = ['inventory', 'billing', 'notifications', 'shipping', 'auth'];
const priorityTypes = ['normal', 'fair'];

const writeStream = fs.createWriteStream(fileName);

const headers = [
    'name', 'code', 'type', 'vnamespace', 'defaultqueuemessagettl',
    'defaultqueuemessagedelaytime', 'queueexpires', 'allowduplicated',
    'maxattempts', 'maxqueuesize', 'prioritytype', 'maxpriority', 'prioritythresholds'
];

// Escribir cabecera (también entre comillas para consistencia)
const quotedHeaders = headers.map(h => `"${h}"`).join(',');
writeStream.write(quotedHeaders + '\n');

for (let i = 1; i <= totalRecords; i++) {
    const vNamespace = namespaces[i % namespaces.length];
    const priorityType = priorityTypes[Math.floor(Math.random() * priorityTypes.length)];
    const maxPriority = priorityType === 'fair' ? Math.floor(Math.random() * 3) + 2 : 1;

    // Generar array de umbrales
    let thresholdsArray = [];
    for (let j = 1; j <= maxPriority; j++) {
        thresholdsArray.push(j * 10);
    }

    // Usar punto y coma (;) en lugar de coma para evitar problemas con el parser CSV
    const thresholdsString = thresholdsArray.join('|');

    const data = {
        name: `q-${vNamespace}-${i}`,
        code: `QUE${i}-${Math.random().toString(36).substring(2, 5).toUpperCase()}`,
        type: 'standard',
        vnamespace: vNamespace,
        defaultqueuemessagettl: 0,
        defaultqueuemessagedelaytime: 0,
        queueexpires: 0,
        allowduplicated: 'true',
        maxattempts: 1,
        maxqueuesize: 0,
        prioritytype: priorityType,
        maxpriority: maxPriority,
        prioritythresholds: thresholdsString
    };

    // Envolver TODOS los campos en comillas para consistencia CSV
    const row = headers.map(h => `"${data[h]}"`).join(',');
    writeStream.write(row + '\n');
}

writeStream.end(() => {
    console.log(`✅ Archivo '${fileName}' generado.`);
    console.log(`Formato CSV estándar con comillas en todos los campos`);
});