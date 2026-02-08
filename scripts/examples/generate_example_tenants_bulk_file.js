const fs = require('fs');

const fileName = 'tenants.csv';
const totalRecords = 1000;

// Listas para generar nombres aleatorios
const prefixes = ['Global', 'Stellar', 'Alpha', 'Nexus', 'Prime', 'Innova', 'Quantum', 'Elite', 'Vertex', 'Omega'];
const middles = ['Data', 'Tech', 'Systems', 'Solutions', 'Dynamics', 'Logic', 'Labs', 'Networks', 'Industries', 'Group'];
const suffixes = ['Corp', 'Inc', 'LLC', 'Enterprises', 'Partners', 'Agency', 'Ventures', 'Co', 'Services', 'Hub'];

// Función para generar un código único de 8 caracteres
function generateCode(index) {
    const chars = 'ABCDEFGHJKLMNPQRSTUVWXYZ23456789'; // Excluimos caracteres ambiguos
    let result = '';
    // Usamos el índice para asegurar unicidad parcial y aleatoriedad
    for (let i = 0; i < 5; i++) {
        result += chars.charAt(Math.floor(Math.random() * chars.length));
    }
    return result + index.toString().padStart(3, '0');
}

const writeStream = fs.createWriteStream(fileName);

// Escribir encabezados
writeStream.write('Name,Code\n');

console.log(`Generando ${totalRecords} registros...`);

for (let i = 1; i <= totalRecords; i++) {
    const name = `${prefixes[i % 10]} ${middles[Math.floor(Math.random() * 10)]} ${suffixes[Math.floor(Math.random() * 10)]}`;
    const code = generateCode(i);

    const row = `"${name}","${code}"\n`;

    // Escribir en el stream
    writeStream.write(row);
}

writeStream.end(() => {
    console.log(`✅ Archivo '${fileName}' creado exitosamente con ${totalRecords} filas.`);
});