const fs = require('fs');

const fileName = 'exchanges.csv';
const totalRecords = 10;

// Datos para la generación aleatoria
const types = ['direct', 'fanout', 'topic', 'headers'];
const namespaces = ['production', 'staging', 'development', 'legacy', 'internal', 'external'];
const regions = ['us-east', 'eu-west', 'ap-south', 'sa-east'];

const writeStream = fs.createWriteStream(fileName);

// Escribir encabezados según tus instrucciones
writeStream.write('ame,Code,Type,VNamespace\n');

console.log(`Generando ${totalRecords} exchanges...`);

for (let i = 1; i <= totalRecords; i++) {
    // Generar Nombre (Ej: us-east-topic-842)
    const type = types[Math.floor(Math.random() * types.length)];
    const region = regions[Math.floor(Math.random() * regions.length)];
    const name = `${region}-${type}-${i}`;

    // Generar Código único (Ej: EX-0001-A9)
    const randomHex = Math.random().toString(16).substring(2, 4).toUpperCase();
    const code = `EX-${i.toString().padStart(4, '0')}-${randomHex}`;

    // Seleccionar Namespace
    const vNamespace = namespaces[Math.floor(Math.random() * namespaces.length)];

    const row = `"${name}","${code}","${type}","${vNamespace}"\n`;

    writeStream.write(row);
}

writeStream.end(() => {
    console.log(`✅ Archivo '${fileName}' creado con éxito.`);
    console.log(`Muestra del formato: Name, Code, Type, VNamespace`);
});