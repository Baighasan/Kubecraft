CREATE TABLE users (
  id SERIAL PRIMARY KEY,
  username VARCHAR(50) UNIQUE NOT NULL,
  email VARCHAR(255) UNIQUE NOT NULL,
  password_hash VARCHAR(255) NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE minecraft_servers (
  id SERIAL PRIMARY KEY,
  user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  server_name VARCHAR(100) NOT NULL,
  namespace VARCHAR(100) UNIQUE NOT NULL,
  minecraft_version VARCHAR(20) DEFAULT '1.21.10',
  game_mode VARCHAR(20) DEFAULT 'survival',
    CHECK (game_mode IN ('survival', 'creative', 'adventure', 'spectator')), 
  max_players INTEGER DEFAULT 20,
    CHECK (max_players >= 1 AND max_players <= 100), 
  difficulty VARCHAR(20) DEFAULT 'normal',
    CHECK (difficulty IN ('peaceful', 'easy', 'normal', 'hard')),
  status VARCHAR(20) DEFAULT 'stopped',
    CHECK (status IN ('stopped', 'starting', 'running', 'stopping', 'error')), 
  server_ip VARCHAR(100),
  server_port INTEGER,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  last_started TIMESTAMP,

  UNIQUE(user_id, server_name)
);

CREATE INDEX idx_user_servers ON minecraft_servers(user_id);
CREATE INDEX idx_status ON minecraft_servers(status);