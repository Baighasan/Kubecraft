import { Pool } from "pg";
import { database } from "../config";

const pool = new Pool({
    user: database.user,
    host: database.host,
    database: database.name,
    password: database.password,
    port: database.port,
    max: 20,
    idleTimeoutMillis: 30000,
    connectionTimeoutMillis: 2000,
})

export const testConnection = async (): Promise<void> => {
    try {
        await query('SELECT 1');
        console.log('✅ Database Connected');
    } catch (error) {
        console.log('❌ Database connection failed:', error);
        throw error;
    }
}

export const query = async (text: string, params?: any[]) => {
    const start = Date.now();
    try {
        const result = await pool.query(text, params);
        const duration = Date.now() - start;
        console.log('Executed query', { text, duration, rows: result.rowCount });
        return result;
    } catch(error) {
        console.error('Query error', { text, error });
        throw error;
    }
}

export default pool;


process.on('SIGINT', async () => {
    console.log('🛑 Shutting down gracefully...');
    await pool.end();
    process.exit(0);
});

process.on('SIGTERM', async () => {
    console.log('🛑 Shutting down gracefully...');
    await pool.end();
    process.exit(0);
});