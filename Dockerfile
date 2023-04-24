FROM ankane/pgvector

RUN echo "CREATE EXTENSION vector;" > /docker-entrypoint-initdb.d/extension.sql