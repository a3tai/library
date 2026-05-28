<!--
@component Lamp — colored status dot.

Four states map to system semantics: `idle`, `running`, `warn`, `fail`.
Active states ride a soft halo (box-shadow ring). Pure presentational
— pair with the surrounding label for assistive tech.

CSS lives in `./lamp.css`.
-->
<script lang="ts">
  import {cn} from '../../../utils/cn';

  type State = 'idle' | 'running' | 'warn' | 'fail';

  type Props = {
    state?: State;
    size?: number;
    class?: string;
  };

  let {state = 'idle', size = 8, class: className}: Props = $props();
</script>

<span
  class={cn('uin-lamp', `uin-lamp-${state}`, className)}
  style="width: {size}px; height: {size}px;"
  aria-hidden="true"
></span>
