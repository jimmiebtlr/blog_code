FROM gitpod/workspace-full

USER root

### Helm3 ###
RUN mkdir -p /tmp/helm/ \
    && curl -fsSL https://get.helm.sh/helm-v3.6.0-linux-amd64.tar.gz | tar -xzvC /tmp/helm/ --strip-components=1 \
    && cp /tmp/helm/helm /usr/local/bin/helm \
    && cp /tmp/helm/helm /usr/local/bin/helm3 \
    && rm -rf /tmp/helm/ \
    && helm completion bash > /usr/share/bash-completion/completions/helm

### kubernetes ###
RUN mkdir -p /usr/local/kubernetes/ && \
    curl -fsSL https://github.com/kubernetes/kubernetes/releases/download/v1.17.16/kubernetes.tar.gz \
    | tar -xzvC /usr/local/kubernetes/ --strip-components=1 \
    && KUBERNETES_SKIP_CONFIRM=true /usr/local/kubernetes/cluster/get-kube-binaries.sh \
    && chown gitpod:gitpod -R /usr/local/kubernetes

ENV PATH=$PATH:/usr/local/kubernetes/cluster/:/usr/local/kubernetes/client/bin/

### kubectl ###
RUN curl -fsSL https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add - \
    # really 'xenial'
    && add-apt-repository -yu "deb https://apt.kubernetes.io/ kubernetes-xenial main" \
    && install-packages kubectl=1.20.0-00 \
    && kubectl completion bash > /usr/share/bash-completion/completions/kubectl

RUN curl -fsSL -o /usr/bin/kubectx https://raw.githubusercontent.com/ahmetb/kubectx/master/kubectx && chmod +x /usr/bin/kubectx \
    && curl -fsSL -o /usr/bin/kubens  https://raw.githubusercontent.com/ahmetb/kubectx/master/kubens  && chmod +x /usr/bin/kubens

# golangci-lint
RUN cd /usr/local && curl -fsSL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s v1.39.0

# https://github.com/gitpod-io/gitpod/blob/master/dev/image/Dockerfile#L118
# Install gcloud
ARG GCS_DIR=/opt/google-cloud-sdk
ENV PATH=$GCS_DIR/bin:$PATH
RUN sudo chown gitpod: /opt \
    && mkdir $GCS_DIR \
    && curl -fsSL https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-sdk-344.0.0-linux-x86_64.tar.gz \
    | tar -xzvC /opt \
    && /opt/google-cloud-sdk/install.sh --quiet --usage-reporting=false --bash-completion=true \
    --additional-components docker-credential-gcr alpha beta \
    # needed for access to our private registries
    && docker-credential-gcr configure-docker && sudo chown -R gitpod: /home/gitpod/.config

# Install tools for gsutil
RUN sudo install-packages \
    gcc \
    python-dev \
    python-setuptools

# Set Application Default Credentials (ADC) based on user-provided env var
RUN echo ". .gitpod/setup-google-adc.sh" >> ~/.bashrc

# Install Terraform
ARG RELEASE_URL="https://releases.hashicorp.com/terraform/0.15.4/terraform_0.15.4_linux_amd64.zip"
RUN mkdir -p ~/.terraform \
    && cd ~/.terraform \
    && curl -fsSL -o terraform_linux_amd64.zip ${RELEASE_URL} \
    && unzip *.zip \
    && rm -f *.zip \
    && printf "terraform -install-autocomplete 2> /dev/null\n" >>~/.bashrc

# Install GraphViz to help debug terraform scripts
RUN sudo install-packages graphviz

ENV PATH=$PATH:$HOME/.terraform