<script lang="ts">
  import type {SettingsDTO} from '../../lib/api/library';
  import Button from '../../lib/components/ui/button/button.svelte';
  import Field from '../../lib/components/ui/field/field.svelte';
  import Input from '../../lib/components/ui/input/input.svelte';
  import NativeSelect from '../../lib/components/ui/native-select/native-select.svelte';

  type Props = {
    current: SettingsDTO | null;
    availableModels: string[];
    url: string;
    embedModel: string;
    chatModel: string;
    apiKey: string;
    saving: boolean;
    testing: boolean;
    onApiKeyChange: () => void;
    onSave: () => void;
    onTestConnection: () => void;
  };

  let {
    current,
    availableModels,
    url = $bindable(''),
    embedModel = $bindable(''),
    chatModel = $bindable(''),
    apiKey = $bindable(''),
    saving,
    testing,
    onApiKeyChange,
    onSave,
    onTestConnection,
  }: Props = $props();

  const modelOptions = $derived(availableModels.map((model) => ({value: model, label: model})));
</script>

<div class="panel">
  <div class="grid">
    <Field label="Base URL" description="OpenAI-compatible base, e.g. http://127.0.0.1:1234/v1">
      {#snippet children({id})}
        <Input {id} bind:value={url} placeholder="http://127.0.0.1:1234/v1" />
      {/snippet}
    </Field>

    <Field
      label="API key"
      description={current?.lmstudioKeyConfigured
        ? 'Stored. Type to replace; leave blank to keep existing.'
        : 'Required if your LM Studio instance has auth enabled.'}
    >
      {#snippet children({id})}
        <Input
          {id}
          type="password"
          bind:value={apiKey}
          oninput={onApiKeyChange}
          placeholder={current?.lmstudioKeyConfigured ? '********' : 'sk-...'}
        />
      {/snippet}
    </Field>

    <Field
      label="Embedding model"
      description={availableModels.length > 0
        ? 'Pick the embedding model LM Studio is serving.'
        : 'LM Studio not reachable. Enter the model id manually or click Test connection.'}
    >
      {#snippet children({id})}
        {#if availableModels.length > 0}
          <NativeSelect
            {id}
            bind:value={embedModel}
            options={[{value: '', label: '- Select -'}, ...modelOptions]}
          />
        {:else}
          <Input {id} bind:value={embedModel} placeholder="harrier-oss-v1-0.6b" />
        {/if}
      {/snippet}
    </Field>

    <Field
      label="Chat model"
      description={availableModels.length > 0
        ? 'Pick the chat model used by the in-app chat panel.'
        : 'LM Studio not reachable. Enter the model id manually or click Test connection.'}
    >
      {#snippet children({id})}
        {#if availableModels.length > 0}
          <NativeSelect
            {id}
            bind:value={chatModel}
            options={[{value: '', label: '- Select -'}, ...modelOptions]}
          />
        {:else}
          <Input {id} bind:value={chatModel} placeholder="lmstudio-community" />
        {/if}
      {/snippet}
    </Field>
  </div>

  <div class="actions">
    <Button variant="primary" disabled={saving} onclick={onSave}>
      {saving ? 'Saving...' : 'Save'}
    </Button>
    <Button variant="outline" disabled={testing} onclick={onTestConnection}>
      {testing ? 'Testing...' : 'Test connection'}
    </Button>
  </div>
</div>

<style>
  .panel {
    display: flex;
    flex-direction: column;
    gap: var(--uin-s-5);
    max-width: 640px;
  }
  .grid {
    display: grid;
    grid-template-columns: 1fr;
    gap: var(--uin-s-4);
  }
  .actions {
    display: flex;
    gap: var(--uin-s-2);
  }
</style>
