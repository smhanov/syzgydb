/**
 * @type {import('vue').ComponentOptions}
 */
export default {
  name: 'Main',
  data() {
    return {
      /** @type {string} */
      message: 'Hello Vue!',
    };
  },
  template: /*html*/`
    <div id="main">
      <p>{{message}}</p>
      <input v-model="message" />
    </div>
  `
}
