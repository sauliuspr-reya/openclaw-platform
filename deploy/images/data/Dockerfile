# OpenClaw Agent - Data Profile
# Data engineering, analytics, and database operations

FROM ghcr.io/openclaw/agent:base AS builder

USER root

# Install Python and database client libraries
RUN apt-get update && apt-get install -y --no-install-recommends \
    python3.11 \
    python3.11-venv \
    python3.11-dev \
    python3-pip \
    build-essential \
    libpq-dev \
    libmysqlclient-dev \
    unixodbc-dev \
    && rm -rf /var/lib/apt/lists/* \
    && apt-get clean

# Create virtual environment
RUN python3.11 -m venv /opt/openclaw/runtime/python

# Install data engineering packages
RUN /opt/openclaw/runtime/python/bin/pip install --no-cache-dir \
    pandas==2.1.* \
    polars==0.19.* \
    duckdb==0.9.* \
    pyarrow==14.* \
    fastparquet==2023.* \
    sqlalchemy==2.0.* \
    psycopg2-binary==2.9.* \
    pymysql==1.1.* \
    pymongo==4.6.* \
    redis==5.0.* \
    elasticsearch==8.* \
    clickhouse-connect==0.6.* \
    dbt-core==1.7.* \
    dbt-postgres==1.7.* \
    great-expectations==0.18.* \
    pandera==0.17.* \
    prefect==2.14.* \
    dagster==1.5.* \
    meltano==3.* \
    singer-sdk==0.34.* \
    boto3==1.33.* \
    google-cloud-bigquery==3.* \
    snowflake-connector-python==3.* \
    delta-spark==3.* \
    pyspark==3.5.* \
    jupyterlab==4.* \
    streamlit==1.29.*

# Install database CLI tools
RUN apt-get update && apt-get install -y --no-install-recommends \
    postgresql-client \
    mysql-client \
    && rm -rf /var/lib/apt/lists/*

# Install DuckDB CLI
RUN curl -fsSL https://github.com/duckdb/duckdb/releases/download/v0.9.2/duckdb_cli-linux-amd64.zip -o /tmp/duckdb.zip \
    && unzip /tmp/duckdb.zip -d /opt/openclaw/tools/bin \
    && rm /tmp/duckdb.zip

# Install ClickHouse client
RUN curl -fsSL https://packages.clickhouse.com/rpm/stable/clickhouse-client-23.10.5.20.x86_64.rpm -o /tmp/clickhouse.rpm \
    && apt-get update && apt-get install -y alien \
    && alien -i /tmp/clickhouse.rpm \
    && rm /tmp/clickhouse.rpm \
    && apt-get remove -y alien \
    && rm -rf /var/lib/apt/lists/*

# Install usql (universal SQL client)
RUN curl -fsSL https://github.com/xo/usql/releases/download/v0.17.5/usql_static-0.17.5-linux-amd64.tar.bz2 | tar xj -C /tmp \
    && mv /tmp/usql_static /opt/openclaw/tools/bin/usql \
    && chmod +x /opt/openclaw/tools/bin/usql

# Fix permissions
RUN chown -R openclaw:openclaw /opt/openclaw

# Final image
FROM ghcr.io/openclaw/agent:base

LABEL org.opencontainers.image.title="OpenClaw Agent - Data"
LABEL org.opencontainers.image.description="OpenClaw agent for data engineering and analytics"
LABEL platform.openclaw.io/profile="data"

# Install runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    postgresql-client \
    default-mysql-client \
    && rm -rf /var/lib/apt/lists/*

# Copy runtimes from builder
COPY --from=builder --chown=openclaw:openclaw /opt/openclaw/runtime /opt/openclaw/runtime
COPY --from=builder --chown=openclaw:openclaw /opt/openclaw/tools /opt/openclaw/tools

# Add Python to PATH
ENV PATH="/opt/openclaw/tools/bin:/opt/openclaw/runtime/python/bin:${PATH}"
ENV PYTHONPATH="/workspace"
ENV PYTHONDONTWRITEBYTECODE=1
ENV PYTHONUNBUFFERED=1

# Profile metadata
ENV OPENCLAW_PROFILE="data"
ENV OPENCLAW_PROFILE_VERSION="1.0.0"

USER openclaw
WORKDIR /workspace
