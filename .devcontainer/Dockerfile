FROM maven:3.9.15-amazoncorretto-25-al2023

# Install Node.js 22
RUN curl -fsSL https://rpm.nodesource.com/setup_22.x | bash - \
    && dnf install -y \
    nodejs \
    git \
    tar \
    gzip \
    unzip \
    zip \
    procps-ng \
    which \
    shadow-utils \
    findutils \
    tmux \
    python3 \
    python3-pip\
    && dnf clean all \
    && curl -fsSL https://getmic.ro | bash \
    && mv micro /usr/bin/micro \
    && npm install -g @openai/codex \
    && npm install -g @fission-ai/openspec@latest

ENV EDITOR=micro
ENV VISUAL=micro

WORKDIR /workspace
CMD ["bash"]
