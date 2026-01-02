<script lang="ts">
  import { onMount } from 'svelte';
  import { getProfiles, getCurrent, saveProfile, switchProfile, deleteProfile, type CurrentStatus } from './lib/api';

  let tool = 'claude';
  let profiles: string[] = [];
  let current: CurrentStatus | null = null;
  let newProfileName = '';
  let loading = false;
  let error = '';
  let refreshSeq = 0;

  async function refresh() {
    const seq = ++refreshSeq;
    const selectedTool = tool;

    loading = true;
    error = '';
    try {
      const [nextProfiles, nextCurrent] = await Promise.all([
        getProfiles(selectedTool),
        getCurrent(selectedTool),
      ]);

      if (seq !== refreshSeq || selectedTool !== tool) return;
      profiles = nextProfiles;
      current = nextCurrent;
    } catch (e) {
      if (seq !== refreshSeq) return;
      error = e instanceof Error ? e.message : 'Failed to load';
    } finally {
      if (seq !== refreshSeq) return;
      loading = false;
    }
  }

  async function handleSwitch(profile: string) {
    const selectedTool = tool;

    loading = true;
    error = '';
    try {
      await switchProfile(selectedTool, profile);
      await refresh();
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to switch';
    } finally {
      loading = false;
    }
  }

  async function handleSave() {
    if (!newProfileName.trim()) return;
    const selectedTool = tool;

    loading = true;
    error = '';
    try {
      await saveProfile(selectedTool, newProfileName.trim());
      newProfileName = '';
      await refresh();
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to save';
    } finally {
      loading = false;
    }
  }

  async function handleDelete(profile: string) {
    if (!confirm(`Delete profile "${profile}"?`)) return;
    const selectedTool = tool;

    loading = true;
    error = '';
    try {
      await deleteProfile(selectedTool, profile);
      await refresh();
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to delete';
    } finally {
      loading = false;
    }
  }

  function selectTool(t: string) {
    tool = t;
    refresh();
  }

  onMount(refresh);
</script>

<main>
  <h1>Tokyo</h1>
  <p class="subtitle">Profile Manager</p>

  <div class="tabs">
    <button class:active={tool === 'claude'} on:click={() => selectTool('claude')}>Claude Code</button>
    <button class:active={tool === 'codex'} on:click={() => selectTool('codex')}>Codex</button>
  </div>

  {#if error}
    <div class="error">{error}</div>
  {/if}

  {#if current}
    <div class="current">
      <span class="label">Current:</span>
      <span class="value" class:modified={current.modified} class:custom={current.custom}>
        {current.custom ? '<custom>' : current.profile}
        {#if current.modified}(modified){/if}
      </span>
    </div>
  {/if}

  <div class="save-form">
    <input
      type="text"
      bind:value={newProfileName}
      placeholder="New profile name"
      on:keydown={(e) => e.key === 'Enter' && handleSave()}
    />
    <button on:click={handleSave} disabled={loading || !newProfileName.trim()}>Save Current</button>
  </div>

  <div class="profiles">
    <h2>Profiles</h2>
    {#if loading}
      <p class="loading">Loading...</p>
    {:else if profiles.length === 0}
      <p class="empty">No profiles saved</p>
    {:else}
      <ul>
        {#each profiles as profile}
          <li class:active={current && !current.custom && current.profile === profile && !current.modified}>
            <span class="name">{profile}</span>
            <div class="actions">
              <button on:click={() => handleSwitch(profile)} disabled={loading}>Switch</button>
              <button class="delete" on:click={() => handleDelete(profile)} disabled={loading}>Delete</button>
            </div>
          </li>
        {/each}
      </ul>
    {/if}
  </div>
</main>

<style>
  main {
    max-width: 500px;
    margin: 0 auto;
    padding: 2rem;
  }

  h1 {
    margin: 0;
    font-size: 2rem;
  }

  .subtitle {
    margin: 0.25rem 0 1.5rem;
    color: #888;
  }

  .tabs {
    display: flex;
    gap: 0.5rem;
    margin-bottom: 1.5rem;
  }

  .tabs button {
    flex: 1;
    padding: 0.75rem;
    background: #2a2a2a;
    border: 1px solid #333;
    color: #888;
  }

  .tabs button.active {
    background: #333;
    color: #fff;
    border-color: #646cff;
  }

  .error {
    background: #ff3e3e22;
    border: 1px solid #ff3e3e;
    color: #ff6b6b;
    padding: 0.75rem;
    border-radius: 4px;
    margin-bottom: 1rem;
  }

  .current {
    background: #2a2a2a;
    padding: 1rem;
    border-radius: 4px;
    margin-bottom: 1rem;
  }

  .current .label {
    color: #888;
  }

  .current .value {
    margin-left: 0.5rem;
    font-weight: 600;
  }

  .current .value.modified {
    color: #f0ad4e;
  }

  .current .value.custom {
    color: #888;
    font-style: italic;
  }

  .save-form {
    display: flex;
    gap: 0.5rem;
    margin-bottom: 1.5rem;
  }

  .save-form input {
    flex: 1;
    padding: 0.75rem;
    background: #1a1a1a;
    border: 1px solid #333;
    border-radius: 4px;
    color: #fff;
  }

  .save-form input:focus {
    outline: none;
    border-color: #646cff;
  }

  .profiles h2 {
    font-size: 1rem;
    color: #888;
    margin: 0 0 0.75rem;
  }

  .profiles ul {
    list-style: none;
    padding: 0;
    margin: 0;
  }

  .profiles li {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 0.75rem;
    background: #2a2a2a;
    border-radius: 4px;
    margin-bottom: 0.5rem;
  }

  .profiles li.active {
    border: 1px solid #646cff;
  }

  .profiles .name {
    font-weight: 500;
  }

  .profiles .actions {
    display: flex;
    gap: 0.5rem;
  }

  .profiles .actions button {
    padding: 0.4rem 0.75rem;
    font-size: 0.85rem;
  }

  .profiles .actions button.delete {
    background: #ff3e3e22;
    border-color: #ff3e3e44;
    color: #ff6b6b;
  }

  .profiles .actions button.delete:hover {
    border-color: #ff3e3e;
  }

  .loading, .empty {
    color: #888;
    text-align: center;
    padding: 2rem;
  }
</style>
