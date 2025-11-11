import dotenv from "dotenv";
dotenv.config();

function requireEnv(key: string): string {
    const value = process.env[key];
    if (!value) {
        throw new Error(`❌ Missing required environment variable: ${key}`);
    }
    return value;
}

export const server = {
  port: Number(process.env.PORT) || 3000,
  env: process.env.NODE_ENV ?? "development",
};

export const database = {
  host: requireEnv("DB_HOST"),
  port: Number(process.env.DB_PORT) || 5432,
  name: requireEnv("DB_NAME"),
  user: requireEnv("DB_USER"),
  password: requireEnv("DB_PASSWORD"),
};