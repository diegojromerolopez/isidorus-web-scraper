#!/bin/bash
set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    CREATE TABLE IF NOT EXISTS scrapings (
        id SERIAL PRIMARY KEY,
        user_id INTEGER,
        url TEXT NOT NULL
    );

    CREATE TABLE IF NOT EXISTS scraped_pages (
        id SERIAL PRIMARY KEY,
        scraping_id INTEGER REFERENCES scrapings(id) ON DELETE CASCADE,
        url TEXT NOT NULL,
        summary TEXT,
        scraped_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
    );

    CREATE TABLE IF NOT EXISTS page_terms (
        id SERIAL PRIMARY KEY,
        scraping_id INTEGER REFERENCES scrapings(id) ON DELETE CASCADE,
        page_id INTEGER REFERENCES scraped_pages(id) ON DELETE CASCADE,
        term TEXT NOT NULL,
        frequency INTEGER DEFAULT 1
    );

    CREATE TABLE IF NOT EXISTS page_images (
        id SERIAL PRIMARY KEY,
        scraping_id INTEGER REFERENCES scrapings(id) ON DELETE CASCADE,
        page_id INTEGER REFERENCES scraped_pages(id) ON DELETE CASCADE,
        image_url TEXT NOT NULL,
        explanation TEXT,
        s3_path TEXT
    );

    CREATE TABLE IF NOT EXISTS page_links (
        id SERIAL PRIMARY KEY,
        scraping_id INTEGER REFERENCES scrapings(id) ON DELETE CASCADE,
        source_page_id INTEGER REFERENCES scraped_pages(id) ON DELETE CASCADE,
        target_url TEXT NOT NULL
    );

    CREATE INDEX IF NOT EXISTS idx_page_terms_term ON page_terms(term);
    CREATE INDEX IF NOT EXISTS idx_scraped_pages_url ON scraped_pages(url);
    
    -- Replication User
    DO \$\$
    BEGIN
        IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'replicator') THEN
            CREATE USER replicator WITH REPLICATION ENCRYPTED PASSWORD 'replicator';
        END IF;
    END
    \$\$;
EOSQL
