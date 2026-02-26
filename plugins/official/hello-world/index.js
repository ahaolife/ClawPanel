/**
 * Hello World - OpenClaw 示例插件
 * 
 * 演示插件基本结构：
 * - activate / deactivate 生命周期
 * - 消息处理
 * - 配置读取
 * - 日志输出
 */

module.exports = {
  name: 'hello-world',
  version: '1.0.0',

  async activate(context) {
    const config = context.getConfig();
    context.log.info('[HelloWorld] 插件已激活');
    context.log.info(`[HelloWorld] 触发关键词: ${config.trigger || '/hello'}`);
    context.log.info(`[HelloWorld] 问候语: ${config.greeting || '你好！'}`);

    context.onMessage(async (msg) => {
      const cfg = context.getConfig();
      if (!cfg.enabled) return;

      const trigger = cfg.trigger || '/hello';
      if (msg.content.trim() === trigger) {
        const greeting = cfg.greeting || '你好！我是 OpenClaw 插件 👋';
        await context.reply(msg, greeting);
        context.log.info(`[HelloWorld] 回复了 ${msg.senderName}: ${greeting}`);
      }
    });
  },

  async deactivate() {
    // 无需清理资源
  },

  async onConfigChange(newConfig, oldConfig) {
    // 配置变更时可以做一些处理
  }
};
