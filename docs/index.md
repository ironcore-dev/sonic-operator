---
# https://vitepress.dev/reference/default-theme-home-page
layout: home

hero:
  name: "Sonic Operator"
  text: "Kubernetes operator for SONiC switches"
  tagline: "Manage Switch Onboarding and Lifecycle using Kubernetes native APIs"
  image:
    src: https://raw.githubusercontent.com/ironcore-dev/ironcore/refs/heads/main/docs/assets/logo_borderless.svg
    alt: IronCore
  actions:
    - theme: brand
      text: Getting started
      link: /quickstart
    - theme: alt
      text: API Reference
      link: /api-reference/api

features:
  - title: Declarative Switch Management
    details: Model switches and their interfaces via CRDs and let controllers reconcile desired state.
  - title: Provisioning Workflows
    details: Serve ZTP scripts and ONIE installer artifacts to bootstrap and upgrade devices.
  - title: Kubernetes Native API
    details: Kubernetes native API for managing switch resources and their lifecycle.
---
