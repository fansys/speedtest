"use strict";

/**
 * LibreSpeed 现代化科技风测速与节点管理引擎
 */

// ================= 多语言 (i18n) 字典系统 =================
const i18nDict = {
  zh: {
    brand_desc: "分布式网络性能测速平台",
    nodes_online_suffix: "个节点在线",
    theme_dark: "暗色",
    theme_light: "亮色",
    sync: "刷新",
    token: "令牌",
    token_configured: "已配置",
    lang_btn: "English",

    node_route: "测速节点",
    direct_node: "中心节点",
    rtt_local: "延迟: 本地",
    rtt_unknown: "延迟: 未知",
    duration_label: "测速时长",
    dur_5: "5 秒 (快速)",
    dur_10: "10 秒",
    dur_15: "15 秒 (默认)",
    dur_20: "20 秒",
    dur_30: "30 秒 (深度)",
    auto_pick: "自动优选节点",

    ping_title: "网络延迟 (Ping)",
    jitter: "抖动",
    min: "最小",
    dl_title: "下载速率",
    ul_title: "上传速率",
    peak: "峰值",
    trans: "流量",

    start_main: "测速",
    start_sub: "点击开始",
    ready: "准备就绪",
    pinging: "正在测试延迟",
    downloading: "正在下载测速",
    uploading: "正在上传测速",
    completed: "测速完成",
    abort: "取消测速",
    retest: "重新测速",

    stream_title: "实时速率波动图",
    legend_dl: "下载",
    legend_ul: "上传",
    peak_tag: "峰值: {val} Mbps",

    pipe_ready: "准备就绪",
    pipe_ping: "网络延迟",
    pipe_dl: "下载测速",
    pipe_ul: "上传测速",
    pipe_done: "测速完成",

    eval_title: "网络质量评估报告",
    eval_done: "测速完成",
    eval_badge_seal: "评估",
    eval_giga_badge: "千兆极速宽带",
    eval_giga_title: "卓越的千兆光纤网络",
    eval_giga_desc: "当前下载速率高达 {val} Mbps，延迟 {ping} ms。支持 8K 超高清秒开与电竞级网络对战。",
    eval_giga_tags: ["8K超清秒开", "千兆极速", "电竞级低延迟", "极速传输"],
    eval_500_badge: "500M 卓越宽带",
    eval_500_title: "高速高品质光纤网络",
    eval_500_desc: "当前下载速率达 {val} Mbps，支持 4K 影视多路并发播放与大文件秒级下载。",
    eval_500_tags: ["4K超清", "游戏联机", "多人共享", "多路并发"],
    eval_100_badge: "百兆高速宽带",
    eval_100_title: "百兆品质光纤宽带",
    eval_100_desc: "当前下载速率 {val} Mbps，满足日常家庭全场景办公、高清视频与多设备上网。",
    eval_100_tags: ["4K影视", "家庭高速", "流畅网课", "云端协同"],
    eval_30_badge: "标准家庭宽带",
    eval_30_title: "常用家庭网络连接",
    eval_30_desc: "当前下载速率 {val} Mbps，满足 1080P 高清观影与日常网页浏览。",
    eval_30_tags: ["1080P流畅", "日常办公", "网页秒开"],
    eval_base_badge: "基础网络连接",
    eval_base_title: "基础网络连接",
    eval_base_desc: "当前下载速率 {val} Mbps，适合基础网页与即时通信。",
    eval_base_tags: ["基础网络", "即时通信"],

    tab_history: "测速历史记录",
    tab_nodes: "节点管理",
    tab_register: "注册新节点",
    export_csv: "导出 CSV",
    clear_history: "清空历史",
    search_nodes: "搜索节点名称或地址...",
    filter_all: "全部",
    filter_online: "在线",
    filter_enabled: "已启用",
    filter_offline: "离线",

    th_time: "时间",
    th_target: "测速目标",
    th_ping: "延迟",
    th_jitter: "抖动",
    th_dl: "下载速率",
    th_ul: "上传速率",
    th_actions: "操作",
    hist_empty: "暂无测速记录，完成测速后将自动记录于此。",

    th_id: "序号",
    th_name: "节点名称",
    th_route: "地址与协议",
    th_fp: "安全指纹",
    th_status: "状态",
    th_enable: "开关",
    th_lat: "最近延迟",
    nodes_empty: "暂无已注册节点",
    nodes_loading: "正在加载节点列表...",
    status_online: "在线",
    status_offline: "离线",
    status_unknown: "未知",
    enabled: "启用",
    disabled: "禁用",
    act_test: "测速",
    act_health: "巡检",
    act_del: "删除",
    act_retest: "重测",

    reg_name_label: "节点显示名称 *",
    reg_name_ph: "例如：上海电信-01",
    reg_addr_label: "节点地址或域名 *",
    reg_addr_ph: "127.0.0.1 或 node.internal",
    reg_port_label: "服务端口 *",
    reg_proto_label: "通信协议",
    reg_meta_label: "元数据 (可选 JSON)",
    reg_meta_ph: '例如：{"region":"cn-east"}',
    reg_hint: "注册成功后服务端将自动生成高熵密钥，并在弹窗中仅展示一次。",
    reg_btn: "立即注册节点",

    key_modal_title: "节点密钥生成成功",
    key_modal_notice: "这是该节点的专属密钥，只会在此处明文展示一次。关闭后系统将无法找回。",
    key_modal_label: "生成的密钥 (NODE_KEY)",
    key_copy_btn: "复制密钥",
    key_copied: "已复制",
    key_modal_hint: "请妥善保存此密钥，并注入节点 Agent 的 NODE_KEY 环境变量。",
    key_modal_close: "我已妥善保存并关闭",

    token_modal_title: "管理与注册令牌配置",
    token_modal_notice: "令牌仅保存在当前浏览器的本地存储中，供管理和注册节点使用。",
    admin_token_label: "管理员令牌 (ADMIN_TOKEN)",
    admin_token_ph: "输入管理员令牌",
    reg_token_label: "节点注册令牌 (REGISTRATION_TOKEN)",
    reg_token_ph: "输入节点注册令牌",
    btn_toggle_pw: "显隐",
    btn_clear_token: "清空本地令牌",
    btn_save_token: "保存配置",
    btn_dismiss: "关闭",

    toast_copied: "密钥已复制到剪贴板",
    toast_copy_fail: "剪贴板不可用，请手动选中复制",
    toast_tokens_saved: "令牌配置已保存到本地",
    toast_tokens_cleared: "已清空本地令牌",
    toast_test_completed: "测速已完成",
    toast_test_aborted: "测速已取消",
    toast_node_deleted: "节点已从矩阵中移除",
    toast_node_registered: "节点注册成功",
    toast_hist_cleared: "已清空测速历史",
    toast_hist_deleted: "已删除该条记录",
    toast_csv_exported: "已导出 CSV 文件",
    toast_duration_set: "测速时长已设置为 {dur} 秒",
    toast_opt_node: "已优选最低延迟节点: {name} ({lat} ms)",
    toast_lang_switched: "Language switched to English",
  },
  en: {
    brand_desc: "DISTRIBUTED NETWORK SPEEDTEST TELEMETRY",
    nodes_online_suffix: "Nodes Online",
    theme_dark: "Dark",
    theme_light: "Light",
    sync: "Refresh",
    token: "Tokens",
    token_configured: "Configured",
    lang_btn: "中文",

    node_route: "TARGET NODE",
    direct_node: "Central Server",
    rtt_local: "Latency: Local",
    rtt_unknown: "Latency: Unknown",
    duration_label: "DURATION",
    dur_5: "5s (Fast)",
    dur_10: "10s",
    dur_15: "15s (Default)",
    dur_20: "20s",
    dur_30: "30s (Deep)",
    auto_pick: "Auto Select Lowest Latency",

    ping_title: "Latency (Ping)",
    jitter: "Jitter",
    min: "Min",
    dl_title: "Download Speed",
    ul_title: "Upload Speed",
    peak: "Peak",
    trans: "Transferred",

    start_main: "START",
    start_sub: "SPEEDTEST",
    ready: "Ready",
    pinging: "Testing Latency...",
    downloading: "Testing Download...",
    uploading: "Testing Upload...",
    completed: "Speedtest Completed",
    abort: "Cancel",
    retest: "Retest",

    stream_title: "Real-time Speed Stream",
    legend_dl: "Download",
    legend_ul: "Upload",
    peak_tag: "Peak: {val} Mbps",

    pipe_ready: "Ready",
    pipe_ping: "Latency",
    pipe_dl: "Download",
    pipe_ul: "Upload",
    pipe_done: "Done",

    eval_title: "Network Quality Report",
    eval_done: "Speedtest Finished",
    eval_badge_seal: "REPORT",
    eval_giga_badge: "Gigabit Fiber",
    eval_giga_title: "Ultra-Fast Gigabit Fiber Connection",
    eval_giga_desc: "Current download speed reached {val} Mbps with {ping} ms latency. Ideal for 8K streaming, pro esports, and massive transfers.",
    eval_giga_tags: ["8K Ultra HD", "Gigabit Speed", "Low Latency", "High Throughput"],
    eval_500_badge: "500M Fiber",
    eval_500_title: "High-Performance 500M Broadband",
    eval_500_desc: "Current download throughput reached {val} Mbps. Supports concurrent 4K streaming and instant downloads.",
    eval_500_tags: ["4K UHD", "Online Gaming", "Multi-Device", "Ultra Smooth"],
    eval_100_badge: "100M Fiber",
    eval_100_title: "100M High-Speed Broadband",
    eval_100_desc: "Current download speed is {val} Mbps. Ideal for home streaming, remote work, and multi-user browsing.",
    eval_100_tags: ["4K Video", "Home High-Speed", "Remote Work", "Cloud Sync"],
    eval_30_badge: "Standard",
    eval_30_title: "Standard Home Broadband",
    eval_30_desc: "Current download speed is {val} Mbps. Smooth for 1080P streaming and general browsing.",
    eval_30_tags: ["1080P HD", "Office Work", "Web Surfing"],
    eval_base_badge: "Basic",
    eval_base_title: "Basic Internet Connection",
    eval_base_desc: "Current download speed is {val} Mbps. Suitable for basic browsing and instant messaging.",
    eval_base_tags: ["Basic Web", "Messaging"],

    tab_history: "Speedtest History",
    tab_nodes: "Node Management",
    tab_register: "Deploy New Node",
    export_csv: "Export CSV",
    clear_history: "Clear History",
    search_nodes: "Search nodes by name or address...",
    filter_all: "All",
    filter_online: "Online",
    filter_enabled: "Enabled",
    filter_offline: "Offline",

    th_time: "Timestamp",
    th_target: "Target Node",
    th_ping: "Latency",
    th_jitter: "Jitter",
    th_dl: "Download",
    th_ul: "Upload",
    th_actions: "Actions",
    hist_empty: "No speedtest records found. Completed tests will be automatically saved here.",

    th_id: "ID",
    th_name: "Node Name",
    th_route: "Address & Protocol",
    th_fp: "Fingerprint",
    th_status: "Status",
    th_enable: "State",
    th_lat: "Latest Latency",
    nodes_empty: "No nodes registered yet",
    nodes_loading: "Loading node list...",
    status_online: "Online",
    status_offline: "Offline",
    status_unknown: "Unknown",
    enabled: "Enabled",
    disabled: "Disabled",
    act_test: "Test",
    act_health: "Inspect",
    act_del: "Delete",
    act_retest: "Retest",

    reg_name_label: "Node Name *",
    reg_name_ph: "e.g. US-West-01",
    reg_addr_label: "Node IP or Hostname *",
    reg_addr_ph: "127.0.0.1 or node.internal",
    reg_port_label: "Port *",
    reg_proto_label: "Protocol",
    reg_meta_label: "Metadata (Optional JSON)",
    reg_meta_ph: 'e.g. {"region":"us-west"}',
    reg_hint: "Server will generate a high-entropy secret key and display it once upon success.",
    reg_btn: "Deploy Node",

    key_modal_title: "Node Key Generated",
    key_modal_notice: "This is the exclusive secret key (NODE_KEY) for this node. It will only be shown once.",
    key_modal_label: "Generated NODE_KEY",
    key_copy_btn: "Copy Key",
    key_copied: "Copied!",
    key_modal_hint: "Please save this key securely and inject it into the Node Agent's NODE_KEY environment variable.",
    key_modal_close: "I have saved this key. Close",

    token_modal_title: "Admin & Registration Tokens",
    token_modal_notice: "Tokens are stored only in local browser localStorage for management API authentication.",
    admin_token_label: "Admin Token (ADMIN_TOKEN)",
    admin_token_ph: "Enter ADMIN_TOKEN",
    reg_token_label: "Registration Token (REGISTRATION_TOKEN)",
    reg_token_ph: "Enter REGISTRATION_TOKEN",
    btn_toggle_pw: "Show/Hide",
    btn_clear_token: "Clear Tokens",
    btn_save_token: "Save Configuration",
    btn_dismiss: "Dismiss",

    toast_copied: "Key copied to clipboard",
    toast_copy_fail: "Clipboard API unavailable, please copy manually",
    toast_tokens_saved: "Tokens saved to local browser",
    toast_tokens_cleared: "Local tokens cleared",
    toast_test_completed: "Speedtest completed successfully",
    toast_test_aborted: "Speedtest aborted",
    toast_node_deleted: "Node removed from matrix",
    toast_node_registered: "Node deployed successfully",
    toast_hist_cleared: "Speedtest history cleared",
    toast_hist_deleted: "Record deleted",
    toast_csv_exported: "CSV exported successfully",
    toast_duration_set: "Speedtest duration set to {dur}s",
    toast_opt_node: "Optimal node selected: {name} ({lat} ms)",
    toast_lang_switched: "界面语言已切换为中文",
  },
};

// ================= 全局状态与 DOM 元素引用 =================
const state = {
  lang: "zh",
  theme: "auto",
  adminToken: "",
  registrationToken: "",
  nodes: [],
  selectedTarget: "direct",
  activeFilter: "all",
  searchQuery: "",
  testDuration: 15,
  isTesting: false,
  abortController: null,
  history: [],
};

const els = {
  // 语言与主题切换
  btnLangToggle: document.getElementById("btn-lang-toggle"),
  langLabel: document.getElementById("lang-label"),
  btnThemeToggle: document.getElementById("btn-theme-toggle"),
  themeIconDark: document.getElementById("theme-icon-dark"),
  themeIconLight: document.getElementById("theme-icon-light"),
  themeLabel: document.getElementById("theme-label"),

  // 导航与统计
  statOnlineCount: document.getElementById("stat-online-count"),
  statTotalCount: document.getElementById("stat-total-count"),
  tokenStatusLabel: document.getElementById("token-status-label"),
  btnRefresh: document.getElementById("btn-refresh"),
  btnOpenTokens: document.getElementById("btn-open-tokens"),

  // 测速节点与时长配置条
  nodeSelect: document.getElementById("speedtest-node-select"),
  durationSelect: document.getElementById("speedtest-duration-select"),
  targetStatusDot: document.getElementById("target-status-dot"),
  targetLatencyBadge: document.getElementById("target-latency-badge"),
  btnPickLowest: document.getElementById("btn-pick-lowest"),

  // 核心实时三大指标
  cardPing: document.getElementById("card-metric-ping"),
  cardDownload: document.getElementById("card-metric-download"),
  cardUpload: document.getElementById("card-metric-upload"),

  valPing: document.getElementById("val-ping"),
  valJitter: document.getElementById("val-jitter"),
  valPingMin: document.getElementById("val-ping-min"),
  valDownload: document.getElementById("val-download"),
  valDownloadPeak: document.getElementById("val-download-peak"),
  valDownloadBytes: document.getElementById("val-download-bytes"),
  valUpload: document.getElementById("val-upload"),
  valUploadPeak: document.getElementById("val-upload-peak"),
  valUploadBytes: document.getElementById("val-upload-bytes"),

  // 仪表盘区域
  gaugeCanvas: document.getElementById("gauge-canvas"),
  dialIdleView: document.getElementById("dial-idle-view"),
  dialRunningView: document.getElementById("dial-running-view"),
  dialFinishedView: document.getElementById("dial-finished-view"),
  btnStartTest: document.getElementById("btn-start-test"),
  btnStartMainText: document.getElementById("btn-start-main-text"),
  btnStartSubText: document.getElementById("btn-start-sub-text"),
  btnCancelTest: document.getElementById("btn-cancel-test"),
  btnRetestCircle: document.getElementById("btn-retest-circle"),
  gaugeLiveValue: document.getElementById("gauge-live-value"),
  gaugeLiveUnit: document.getElementById("gauge-live-unit"),
  gaugeLiveLabel: document.getElementById("gauge-live-label"),
  finishRatingBadge: document.getElementById("finish-rating-badge"),
  finishScoreDesc: document.getElementById("finish-score-desc"),

  // 实时示波器速度图表
  chartCanvas: document.getElementById("chart-canvas"),
  chartPeakBadge: document.getElementById("chart-peak-badge"),

  // 步骤流水线进度条
  meterBar: document.getElementById("test-meter-bar"),

  // 质量评估
  ratingEvalCard: document.getElementById("rating-evaluation-card"),
  evalTitle: document.getElementById("eval-title"),
  evalDesc: document.getElementById("eval-desc"),
  evalTags: document.getElementById("eval-tags"),

  // 状态横幅
  statusBanner: document.getElementById("test-status-banner"),
  statusText: document.getElementById("test-status-text"),
  errorBanner: document.getElementById("test-error-banner"),
  errorText: document.getElementById("test-error-text"),
  btnDismissError: document.getElementById("btn-dismiss-error"),

  // 选项卡系统
  tabBtns: document.querySelectorAll(".hud-tab"),
  tabPanes: document.querySelectorAll(".hud-tab-pane"),
  tabActionsHistory: document.getElementById("tab-actions-history"),
  tabActionsNodes: document.getElementById("tab-actions-nodes"),

  // 历史记录
  historyCountBadge: document.getElementById("history-count-badge"),
  historyTableBody: document.getElementById("history-table-body"),
  historyEmpty: document.getElementById("history-empty"),
  btnExportHistory: document.getElementById("btn-export-history"),
  btnClearHistory: document.getElementById("btn-clear-history"),

  // 节点管理
  nodesCountBadge: document.getElementById("nodes-count-badge"),
  filterSearch: document.getElementById("filter-search"),
  filterPills: document.querySelectorAll(".hud-filter-btn"),
  nodesTableBody: document.getElementById("nodes-table-body"),
  nodesLoading: document.getElementById("nodes-loading"),
  nodesEmpty: document.getElementById("nodes-empty"),
  emptyMessage: document.getElementById("empty-message"),

  // 节点注册表单
  formRegister: document.getElementById("form-register"),
  regName: document.getElementById("reg-name"),
  regAddress: document.getElementById("reg-address"),
  regPort: document.getElementById("reg-port"),
  regProtocol: document.getElementById("reg-protocol"),
  regMetadata: document.getElementById("reg-metadata"),

  // 模态弹窗
  modalOneTimeKey: document.getElementById("modal-one-time-key"),
  displayNewKey: document.getElementById("display-new-key"),
  btnCopyKey: document.getElementById("btn-copy-key"),
  copyKeyText: document.getElementById("copy-key-text"),
  btnCloseKeyModal: document.getElementById("btn-close-key-modal"),

  modalTokens: document.getElementById("modal-tokens"),
  formTokens: document.getElementById("form-tokens"),
  inputAdminToken: document.getElementById("input-admin-token"),
  inputRegToken: document.getElementById("input-registration-token"),
  btnCloseTokens: document.getElementById("btn-close-tokens"),
  btnClearTokens: document.getElementById("btn-clear-tokens"),

  toastContainer: document.getElementById("toast-container"),
};

// ================= 多语言系统实现 =================

function t(key, params = {}) {
  const dict = i18nDict[state.lang] || i18nDict.zh;
  let str = dict[key] || (i18nDict.zh[key] || key);
  for (const [k, v] of Object.entries(params)) {
    str = str.replace(new RegExp(`\\{${k}\\}`, "g"), v);
  }
  return str;
}

function initLanguage() {
  const saved = localStorage.getItem("ls_lang");
  if (saved === "zh" || saved === "en") {
    state.lang = saved;
  } else {
    const navLang = (navigator.language || (navigator.languages && navigator.languages[0]) || "").toLowerCase();
    state.lang = navLang.startsWith("zh") ? "zh" : "en";
  }
  applyLanguage(state.lang, false);
}

function setLanguage(lang) {
  state.lang = lang === "en" ? "en" : "zh";
  localStorage.setItem("ls_lang", state.lang);
  applyLanguage(state.lang, true);
  showToast(t("toast_lang_switched"), "info");
}

function toggleLanguage() {
  const next = state.lang === "zh" ? "en" : "zh";
  setLanguage(next);
}

function applyLanguage(lang, updateDynamic = true) {
  document.documentElement.lang = lang === "zh" ? "zh-CN" : "en";

  els.langLabel.textContent = t("lang_btn");

  document.querySelectorAll("[data-i18n]").forEach((el) => {
    const key = el.getAttribute("data-i18n");
    if (key) el.textContent = t(key);
  });

  document.querySelectorAll("[data-i18n-ph]").forEach((el) => {
    const key = el.getAttribute("data-i18n-ph");
    if (key) el.setAttribute("placeholder", t(key));
  });

  document.querySelectorAll("[data-i18n-title]").forEach((el) => {
    const key = el.getAttribute("data-i18n-title");
    if (key) el.setAttribute("title", t(key));
  });

  updateTokenUI();

  if (updateDynamic) {
    renderNodeSelect();
    renderNodesTable();
    renderHistory();
    if (gaugeEngine) gaugeEngine.draw();
    if (chartEngine) chartEngine.draw();
  }
}

// ================= HiDPI 高清渲染辅助函数 =================

function resizeCanvasHiDPI(canvas, ctx, defaultW, defaultH) {
  const dpr = window.devicePixelRatio || 1;
  const rect = canvas.getBoundingClientRect();
  const displayW = Math.round(rect.width || defaultW);
  const displayH = Math.round(rect.height || defaultH);

  canvas.width = Math.round(displayW * dpr);
  canvas.height = Math.round(displayH * dpr);
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0);

  return { w: displayW, h: displayH };
}

// ================= 科技风 HUD 环形反应堆仪表盘 =================

class SpeedometerGauge {
  constructor(canvas) {
    this.canvas = canvas;
    this.ctx = canvas.getContext("2d");
    this.w = 320;
    this.h = 320;
    this.value = 0;
    this.targetValue = 0;
    this.maxScale = 100;
    this.mode = "idle";
    this.animFrame = null;
    this.unit = "Mbps";
    this.label = t("ready");
    this.resize();
    this.startAnimation();
  }

  resize() {
    const size = resizeCanvasHiDPI(this.canvas, this.ctx, 320, 320);
    this.w = size.w;
    this.h = size.h;
    this.draw();
  }

  setMode(mode, unit = "Mbps", label = "") {
    this.mode = mode;
    this.unit = unit;
    this.label = label || t("ready");
    this.targetValue = 0;
    this.value = 0;
    this.updateScale();
    this.updateCenterView();
  }

  setValue(val, unit = null, label = null) {
    this.targetValue = Math.max(0, val);
    if (unit) this.unit = unit;
    if (label) this.label = label;
    this.updateScale();
  }

  updateScale() {
    if (this.mode === "ping") {
      this.maxScale = this.targetValue > 200 ? 500 : 200;
    } else {
      if (this.targetValue > 500) this.maxScale = 1000;
      else if (this.targetValue > 250) this.maxScale = 500;
      else if (this.targetValue > 100) this.maxScale = 250;
      else if (this.targetValue > 50) this.maxScale = 100;
      else this.maxScale = 50;
    }
  }

  updateCenterView() {
    if (this.mode === "idle") {
      els.dialIdleView.classList.remove("hidden");
      els.dialRunningView.classList.add("hidden");
      els.dialFinishedView.classList.add("hidden");
    } else if (this.mode === "finished") {
      els.dialIdleView.classList.add("hidden");
      els.dialRunningView.classList.add("hidden");
      els.dialFinishedView.classList.remove("hidden");
    } else {
      els.dialIdleView.classList.add("hidden");
      els.dialRunningView.classList.remove("hidden");
      els.dialFinishedView.classList.add("hidden");
      els.gaugeLiveLabel.textContent = this.label;
      els.gaugeLiveUnit.textContent = this.unit;
    }
  }

  startAnimation() {
    const loop = () => {
      const diff = this.targetValue - this.value;
      this.value += diff * 0.12;
      if (Math.abs(diff) < 0.02) {
        this.value = this.targetValue;
      }

      this.draw();

      if (this.mode !== "idle" && this.mode !== "finished") {
        const formattedNum = this.value >= 100 ? this.value.toFixed(1) : this.value.toFixed(2);
        els.gaugeLiveValue.textContent = formattedNum;
        els.gaugeLiveUnit.textContent = this.unit;
        els.gaugeLiveLabel.textContent = this.label;

        if (this.mode === "download") {
          els.valDownload.textContent = formattedNum;
        } else if (this.mode === "upload") {
          els.valUpload.textContent = formattedNum;
        } else if (this.mode === "ping") {
          els.valPing.textContent = this.value.toFixed(1);
        }
      }

      this.animFrame = requestAnimationFrame(loop);
    };
    this.animFrame = requestAnimationFrame(loop);
  }

  draw() {
    const ctx = this.ctx;
    const w = this.w;
    const h = this.h;
    const cx = w / 2;
    const cy = h / 2;
    const r = w / 2 - 24;

    ctx.clearRect(0, 0, w, h);

    const startAngle = 0.72 * Math.PI;
    const endAngle = 2.28 * Math.PI;
    const sweep = endAngle - startAngle;

    const isDark = document.documentElement.getAttribute("data-theme") !== "light";

    // 1. 发光环轨
    ctx.lineWidth = 7;
    ctx.lineCap = "round";
    ctx.strokeStyle = isDark ? "rgba(0, 240, 255, 0.09)" : "rgba(0, 136, 255, 0.12)";
    ctx.beginPath();
    ctx.arc(cx, cy, r, startAngle, endAngle);
    ctx.stroke();

    // 2. 环形刻度
    const ticks = 8;
    ctx.font = "700 10px monospace";
    ctx.textAlign = "center";
    ctx.textBaseline = "middle";

    for (let i = 0; i <= ticks; i++) {
      const frac = i / ticks;
      const angle = startAngle + frac * sweep;
      const innerR = r - 10;
      const outerR = r - 4;
      const textR = r - 20;

      const x1 = cx + Math.cos(angle) * innerR;
      const y1 = cy + Math.sin(angle) * innerR;
      const x2 = cx + Math.cos(angle) * outerR;
      const y2 = cy + Math.sin(angle) * outerR;

      ctx.lineWidth = i % 2 === 0 ? 2 : 1;
      ctx.strokeStyle = isDark ? "rgba(0, 240, 255, 0.28)" : "rgba(0, 136, 255, 0.35)";
      ctx.beginPath();
      ctx.moveTo(x1, y1);
      ctx.lineTo(x2, y2);
      ctx.stroke();

      if (i % 2 === 0) {
        const tx = cx + Math.cos(angle) * textR;
        const ty = cy + Math.sin(angle) * textR;
        const scaleVal = Math.round(this.maxScale * frac);
        ctx.fillStyle = isDark ? "#64748b" : "#94a3b8";
        ctx.fillText(String(scaleVal), tx, ty);
      }
    }

    // 3. 动态激光测速弧线
    if (this.value > 0) {
      const progress = Math.min(1, this.value / this.maxScale);
      const currentAngle = startAngle + progress * sweep;

      let grad = ctx.createLinearGradient(0, cy - r, w, cy + r);
      if (this.mode === "ping") {
        grad.addColorStop(0, "#ffb800");
        grad.addColorStop(1, "#ffd566");
      } else if (this.mode === "download") {
        grad.addColorStop(0, "#00f0ff");
        grad.addColorStop(1, "#3b82f6");
      } else if (this.mode === "upload") {
        grad.addColorStop(0, "#b026ff");
        grad.addColorStop(1, "#ff3366");
      } else {
        grad.addColorStop(0, "#00f0ff");
        grad.addColorStop(1, "#00ff88");
      }

      ctx.lineWidth = 9;
      ctx.lineCap = "round";
      ctx.strokeStyle = grad;
      ctx.beginPath();
      ctx.arc(cx, cy, r, startAngle, currentAngle);
      ctx.stroke();

      // 4. 激光发光顶点
      const headX = cx + Math.cos(currentAngle) * r;
      const headY = cy + Math.sin(currentAngle) * r;

      ctx.fillStyle = "#ffffff";
      ctx.beginPath();
      ctx.arc(headX, headY, 5, 0, Math.PI * 2);
      ctx.fill();

      ctx.shadowColor = this.mode === "upload" ? "#b026ff" : "#00f0ff";
      ctx.shadowBlur = 12;
      ctx.beginPath();
      ctx.arc(headX, headY, 6, 0, Math.PI * 2);
      ctx.stroke();
      ctx.shadowBlur = 0;
    }
  }
}

// ================= 高清实时示波器图表 =================

class SpeedChart {
  constructor(canvas) {
    this.canvas = canvas;
    this.ctx = canvas.getContext("2d");
    this.w = 580;
    this.h = 180;
    this.downloadPoints = [];
    this.uploadPoints = [];
    this.peakSpeed = 0;
    this.maxScale = 50;
    this.maxTime = 15;
    this.resize();
    this.reset();
  }

  resize() {
    const size = resizeCanvasHiDPI(this.canvas, this.ctx, 580, 180);
    this.w = size.w;
    this.h = size.h;
    this.draw();
  }

  reset() {
    this.downloadPoints = [];
    this.uploadPoints = [];
    this.peakSpeed = 0;
    this.maxScale = 50;
    this.maxTime = state.testDuration || 15;
    els.chartPeakBadge.textContent = t("peak_tag", { val: "--" });
    this.draw();
  }

  addPoint(type, timeSec, mbps) {
    const pt = { time: timeSec, val: Math.max(0, mbps) };
    if (type === "download") {
      this.downloadPoints.push(pt);
    } else if (type === "upload") {
      this.uploadPoints.push(pt);
    }

    if (mbps > this.peakSpeed) {
      this.peakSpeed = mbps;
      els.chartPeakBadge.textContent = t("peak_tag", { val: this.peakSpeed.toFixed(1) });
    }

    if (this.peakSpeed > this.maxScale) {
      if (this.peakSpeed > 500) this.maxScale = 1000;
      else if (this.peakSpeed > 250) this.maxScale = 500;
      else if (this.peakSpeed > 100) this.maxScale = 250;
      else if (this.peakSpeed > 50) this.maxScale = 100;
      else this.maxScale = 50;
    }

    this.draw();
  }

  draw() {
    const ctx = this.ctx;
    const w = this.w;
    const h = this.h;
    const padding = { top: 15, right: 20, bottom: 22, left: 45 };
    const chartW = w - padding.left - padding.right;
    const chartH = h - padding.top - padding.bottom;

    ctx.clearRect(0, 0, w, h);

    const isDark = document.documentElement.getAttribute("data-theme") !== "light";

    // 1. 示波器网格线与 Y 轴刻度
    const yTicks = 3;
    ctx.lineWidth = 1;
    ctx.strokeStyle = isDark ? "rgba(0, 240, 255, 0.08)" : "rgba(0, 136, 255, 0.1)";
    ctx.fillStyle = isDark ? "#64748b" : "#94a3b8";
    ctx.font = "10px monospace";
    ctx.textAlign = "right";
    ctx.textBaseline = "middle";

    for (let i = 0; i <= yTicks; i++) {
      const frac = i / yTicks;
      const y = padding.top + chartH * (1 - frac);
      const val = Math.round(this.maxScale * frac);

      ctx.beginPath();
      ctx.moveTo(padding.left, y);
      ctx.lineTo(w - padding.right, y);
      ctx.stroke();

      ctx.fillText(`${val}`, padding.left - 6, y);
    }

    // 2. 时间 X 轴网格线
    const maxT = this.maxTime || 15;
    const step = maxT <= 10 ? 2 : maxT <= 20 ? 5 : 10;
    ctx.textAlign = "center";
    ctx.textBaseline = "top";
    for (let s = 0; s <= maxT; s += step) {
      const x = padding.left + (s / maxT) * chartW;
      ctx.fillText(`${s}s`, x, h - padding.bottom + 6);
    }

    // 3. 绘制平滑激光波形曲线
    const drawSeries = (points, strokeColor, fillColor) => {
      if (points.length < 2) return;

      const coords = points.map((p) => ({
        x: padding.left + Math.min(1, p.time / maxT) * chartW,
        y: padding.top + chartH * (1 - Math.min(1, p.val / this.maxScale)),
      }));

      ctx.beginPath();
      ctx.moveTo(coords[0].x, coords[0].y);
      for (let i = 0; i < coords.length - 1; i++) {
        const xc = (coords[i].x + coords[i + 1].x) / 2;
        const yc = (coords[i].y + coords[i + 1].y) / 2;
        ctx.quadraticCurveTo(coords[i].x, coords[i].y, xc, yc);
      }
      ctx.lineTo(coords[coords.length - 1].x, coords[coords.length - 1].y);
      ctx.lineTo(coords[coords.length - 1].x, padding.top + chartH);
      ctx.lineTo(coords[0].x, padding.top + chartH);
      ctx.closePath();

      const grad = ctx.createLinearGradient(0, padding.top, 0, padding.top + chartH);
      grad.addColorStop(0, fillColor);
      grad.addColorStop(1, "rgba(0, 0, 0, 0)");
      ctx.fillStyle = grad;
      ctx.fill();

      ctx.beginPath();
      ctx.moveTo(coords[0].x, coords[0].y);
      for (let i = 0; i < coords.length - 1; i++) {
        const xc = (coords[i].x + coords[i + 1].x) / 2;
        const yc = (coords[i].y + coords[i + 1].y) / 2;
        ctx.quadraticCurveTo(coords[i].x, coords[i].y, xc, yc);
      }
      ctx.lineTo(coords[coords.length - 1].x, coords[coords.length - 1].y);
      ctx.lineWidth = 2.4;
      ctx.strokeStyle = strokeColor;
      ctx.stroke();
    };

    drawSeries(this.downloadPoints, "#00f0ff", "rgba(0, 240, 255, 0.22)");
    drawSeries(this.uploadPoints, "#b026ff", "rgba(176, 38, 255, 0.22)");
  }
}

let gaugeEngine = null;
let chartEngine = null;

// ================= 网络质量评级计算 =================

function evaluateBroadband(downloadMbps, pingMs) {
  let title = t("eval_title");
  let desc = "";
  let badge = t("eval_giga_badge");
  let tags = [];

  const valStr = downloadMbps.toFixed(1);
  const pingStr = pingMs.toFixed(1);

  if (downloadMbps >= 500) {
    badge = t("eval_giga_badge");
    title = t("eval_giga_title");
    desc = t("eval_giga_desc", { val: valStr, ping: pingStr });
    tags = t("eval_giga_tags");
  } else if (downloadMbps >= 200) {
    badge = t("eval_500_badge");
    title = t("eval_500_title");
    desc = t("eval_500_desc", { val: valStr, ping: pingStr });
    tags = t("eval_500_tags");
  } else if (downloadMbps >= 100) {
    badge = t("eval_100_badge");
    title = t("eval_100_title");
    desc = t("eval_100_desc", { val: valStr, ping: pingStr });
    tags = t("eval_100_tags");
  } else if (downloadMbps >= 30) {
    badge = t("eval_30_badge");
    title = t("eval_30_title");
    desc = t("eval_30_desc", { val: valStr, ping: pingStr });
    tags = t("eval_30_tags");
  } else {
    badge = t("eval_base_badge");
    title = t("eval_base_title");
    desc = t("eval_base_desc", { val: valStr, ping: pingStr });
    tags = t("eval_base_tags");
  }

  els.finishRatingBadge.textContent = badge;
  els.finishScoreDesc.textContent = title;
  els.evalTitle.textContent = title;
  els.evalDesc.textContent = desc;

  if (Array.isArray(tags)) {
    els.evalTags.innerHTML = tags.map((tg) => `<span class="tag-eval">${escapeHtml(tg)}</span>`).join("");
  }
  els.ratingEvalCard.classList.remove("hidden");
}

// ================= 测速历史持久化管理 =================

function loadHistory() {
  try {
    const raw = localStorage.getItem("ls_history");
    state.history = raw ? JSON.parse(raw) : [];
  } catch {
    state.history = [];
  }
  renderHistory();
}

function saveHistoryRecord(record) {
  state.history.unshift(record);
  if (state.history.length > 50) {
    state.history = state.history.slice(0, 50);
  }
  localStorage.setItem("ls_history", JSON.stringify(state.history));
  renderHistory();
}

function clearAllHistory() {
  if (!confirm(t("clear_history") + "?")) return;
  state.history = [];
  localStorage.removeItem("ls_history");
  renderHistory();
  showToast(t("toast_hist_cleared"), "info");
}

function deleteHistoryItem(id) {
  state.history = state.history.filter((h) => h.id !== id);
  localStorage.setItem("ls_history", JSON.stringify(state.history));
  renderHistory();
  showToast(t("toast_hist_deleted"), "info");
}

function exportHistoryCSV() {
  if (state.history.length === 0) {
    showToast(t("hist_empty"), "info");
    return;
  }
  let csv = `${t("th_time")},${t("th_target")},${t("th_ping")} (ms),${t("th_jitter")} (ms),${t("th_dl")} (Mbps),${t("th_ul")} (Mbps)\n`;
  state.history.forEach((h) => {
    csv += `"${h.timestamp}","${h.targetName}",${h.ping},${h.jitter},${h.download},${h.upload}\n`;
  });

  const blob = new Blob(["\ufeff" + csv], { type: "text/csv;charset=utf-8;" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = `librespeed-history-${new Date().toISOString().slice(0, 10)}.csv`;
  a.click();
  URL.revokeObjectURL(url);
  showToast(t("toast_csv_exported"), "success");
}

function renderHistory() {
  const tbody = els.historyTableBody;
  tbody.innerHTML = "";

  els.historyCountBadge.textContent = String(state.history.length);

  if (state.history.length === 0) {
    els.historyEmpty.classList.remove("hidden");
    return;
  }

  els.historyEmpty.classList.add("hidden");

  state.history.forEach((item) => {
    const tr = document.createElement("tr");
    tr.innerHTML = `
      <td><span style="font-family:var(--font-mono);font-size:0.8rem;">${escapeHtml(item.timestamp)}</span></td>
      <td><strong>${escapeHtml(item.targetName)}</strong></td>
      <td><span class="val-ping">${item.ping.toFixed(1)} ms</span></td>
      <td><span>${item.jitter.toFixed(1)} ms</span></td>
      <td><span class="val-dl">${item.download.toFixed(2)} Mbps</span></td>
      <td><span class="val-ul">${item.upload.toFixed(2)} Mbps</span></td>
      <td class="col-right">
        <button class="hud-action-link" data-action="retest" data-target="${escapeHtml(item.targetId)}" title="${t("act_retest")}">${t("act_retest")}</button>
        <button class="hud-danger-link" data-action="del-hist" data-id="${escapeHtml(item.id)}" title="${t("act_del")}">${t("act_del")}</button>
      </td>
    `;
    tbody.appendChild(tr);
  });
}

// ================= 主题管理 (亮色 / 暗色，默认自动检测系统) =================

function initTheme() {
  const savedTheme = localStorage.getItem("ls_theme");
  if (savedTheme === "light" || savedTheme === "dark") {
    setTheme(savedTheme, false);
  } else {
    const isDark = window.matchMedia("(prefers-color-scheme: dark)").matches;
    setTheme(isDark ? "dark" : "light", false);
  }

  window.matchMedia("(prefers-color-scheme: dark)").addEventListener("change", (e) => {
    if (!localStorage.getItem("ls_theme")) {
      setTheme(e.matches ? "dark" : "light", false);
    }
  });
}

function setTheme(theme, save = true) {
  state.theme = theme === "light" ? "light" : "dark";
  if (save) {
    localStorage.setItem("ls_theme", state.theme);
  }
  applyThemeToDOM(state.theme);
  if (chartEngine) chartEngine.draw();
}

function applyThemeToDOM(theme) {
  if (theme === "light") {
    document.documentElement.setAttribute("data-theme", "light");
    els.themeIconDark.classList.remove("hidden");
    els.themeIconLight.classList.add("hidden");
    els.themeLabel.textContent = t("theme_dark");
  } else {
    document.documentElement.setAttribute("data-theme", "dark");
    els.themeIconDark.classList.add("hidden");
    els.themeIconLight.classList.remove("hidden");
    els.themeLabel.textContent = t("theme_light");
  }
}

function toggleTheme() {
  const next = state.theme === "dark" ? "light" : "dark";
  setTheme(next, true);
}

// ================= 辅助工具函数 =================

function showToast(message, type = "info") {
  const toast = document.createElement("div");
  toast.className = `toast toast-${type}`;
  toast.textContent = message;
  els.toastContainer.appendChild(toast);
  setTimeout(() => {
    toast.style.opacity = "0";
    toast.style.transform = "translateY(8px)";
    toast.style.transition = "all 0.3s ease";
    setTimeout(() => toast.remove(), 300);
  }, 3000);
}

function escapeHtml(str) {
  if (str == null) return "";
  const div = document.createElement("div");
  div.textContent = String(str);
  return div.innerHTML;
}

function formatBytes(bytes) {
  if (bytes === 0 || !bytes) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
}

// ================= 令牌存储与请求头 =================

function loadTokens() {
  state.adminToken = localStorage.getItem("ls_admin_token") || "";
  state.registrationToken = localStorage.getItem("ls_registration_token") || "";
  updateTokenUI();
}

function saveTokens(admin, reg) {
  state.adminToken = admin.trim();
  state.registrationToken = reg.trim();
  if (state.adminToken) localStorage.setItem("ls_admin_token", state.adminToken);
  else localStorage.removeItem("ls_admin_token");
  if (state.registrationToken) localStorage.setItem("ls_registration_token", state.registrationToken);
  else localStorage.removeItem("ls_registration_token");
  updateTokenUI();
  showToast(t("toast_tokens_saved"), "success");
}

function clearTokens() {
  state.adminToken = "";
  state.registrationToken = "";
  localStorage.removeItem("ls_admin_token");
  localStorage.removeItem("ls_registration_token");
  els.inputAdminToken.value = "";
  els.inputRegToken.value = "";
  updateTokenUI();
  showToast(t("toast_tokens_cleared"), "info");
}

function updateTokenUI() {
  if (state.adminToken || state.registrationToken) {
    els.tokenStatusLabel.textContent = t("token_configured");
  } else {
    els.tokenStatusLabel.textContent = t("token");
  }
}

function adminHeaders() {
  const headers = {};
  if (state.adminToken) {
    headers["X-Admin-Token"] = state.adminToken;
  }
  return headers;
}

function registrationHeaders() {
  const headers = {};
  if (state.registrationToken) {
    headers["X-Registration-Token"] = state.registrationToken;
  }
  return headers;
}

async function apiFetch(path, { method = "GET", headers = {}, body, signal } = {}) {
  const reqHeaders = { "Content-Type": "application/json", ...headers };
  const resp = await fetch(path, {
    method,
    headers: reqHeaders,
    body: body !== undefined ? JSON.stringify(body) : undefined,
    signal,
  });

  let data = null;
  const text = await resp.text();
  if (text) {
    try {
      data = JSON.parse(text);
    } catch {
      data = text;
    }
  }

  if (!resp.ok) {
    const detail = (data && data.detail) || resp.statusText;
    const msg = typeof detail === "string" ? detail : JSON.stringify(detail);
    const err = new Error(msg);
    err.status = resp.status;
    err.data = data;
    throw err;
  }
  return data;
}

// ================= LibreSpeed 测速引擎 =================

class SpeedtestEngine {
  constructor() {
    this.target = "direct";
    this.nodeObj = null;
    this.abortController = null;
    this.pingCount = 6;
    this.duration = 15;
  }

  async run(targetId) {
    this.target = targetId;
    this.duration = state.testDuration || 15;
    this.abortController = new AbortController();
    state.abortController = this.abortController;

    if (this.target !== "direct") {
      this.nodeObj = state.nodes.find((n) => String(n.id) === String(targetId));
    } else {
      this.nodeObj = null;
    }

    this.setUIStage("ready", 0);
    this.resetMetrics();
    els.ratingEvalCard.classList.add("hidden");
    if (chartEngine) {
      chartEngine.maxTime = this.duration;
      chartEngine.reset();
    }
    if (gaugeEngine) gaugeEngine.setMode("idle", "Mbps", t("ready"));
    this.showStatus(t("ready") + "...");

    const testSummary = {
      id: "hist-" + Date.now(),
      timestamp: new Date().toLocaleString(state.lang === "zh" ? "zh-CN" : "en-US", { hour12: false }),
      targetId: this.target,
      targetName: this.nodeObj ? this.nodeObj.name : t("direct_node"),
      ping: 0,
      jitter: 0,
      download: 0,
      downloadBytes: 0,
      upload: 0,
      uploadBytes: 0,
    };

    try {
      // 阶段 1: Ping 与 Jitter 测试
      const pingRes = await this.runPingPhase();
      testSummary.ping = pingRes.avg;
      testSummary.jitter = pingRes.jitter;

      // 阶段 2: 流式下载测速
      const dlRes = await this.runDownloadPhase();
      testSummary.download = dlRes.mbps;
      testSummary.downloadBytes = dlRes.bytes;

      // 阶段 3: 上传测速
      const ulRes = await this.runUploadPhase();
      testSummary.upload = ulRes.mbps;
      testSummary.uploadBytes = ulRes.bytes;

      // 阶段 4: 完成
      this.setUIStage("finished", 100);
      if (gaugeEngine) gaugeEngine.setMode("finished", "Mbps", t("completed"));
      this.showStatus(t("completed"));
      showToast(t("toast_test_completed"), "success");

      // 评估宽带与网络质量
      evaluateBroadband(testSummary.download, testSummary.ping);

      // 自动保存测速历史
      saveHistoryRecord(testSummary);

      if (this.nodeObj) {
        refreshNodes(true);
      }
    } catch (err) {
      if (err.name === "AbortError" || err.message === "aborted") {
        this.showStatus(t("toast_test_aborted"));
        showToast(t("toast_test_aborted"), "info");
      } else {
        console.error("Speedtest error:", err);
        this.showError(`${err.message}`);
        showToast(`${err.message}`, "error");
      }
      if (gaugeEngine) gaugeEngine.setMode("idle", "Mbps", "ABORTED");
    } finally {
      this.finishTest();
    }
  }

  abort() {
    if (this.abortController) {
      this.abortController.abort();
    }
  }

  async runPingPhase() {
    this.setUIStage("ping", 15);
    els.cardPing.classList.add("active");
    if (gaugeEngine) gaugeEngine.setMode("ping", "ms", t("pinging"));
    this.showStatus(t("pinging") + "...");

    const pings = [];
    const signal = this.abortController.signal;

    for (let i = 0; i < this.pingCount; i++) {
      if (signal.aborted) throw new Error("aborted");

      const pingURL = this.target === "direct"
        ? "/api/speedtest/ping"
        : `/api/nodes/${this.target}/ping`;

      const t0 = performance.now();
      try {
        const resp = await fetch(pingURL, {
          method: "GET",
          headers: { ...adminHeaders(), "Cache-Control": "no-cache" },
          signal,
        });
        if (!resp.ok && resp.status !== 204) {
          throw new Error(`HTTP ${resp.status}`);
        }
      } catch (err) {
        if (err.name === "AbortError") throw err;
        throw new Error(`${err.message}`);
      }
      const latency = performance.now() - t0;
      pings.push(latency);

      const min = Math.min(...pings);
      const avg = pings.reduce((a, b) => a + b, 0) / pings.length;

      let jitter = 0;
      if (pings.length > 1) {
        let diffSum = 0;
        for (let j = 1; j < pings.length; j++) {
          diffSum += Math.abs(pings[j] - pings[j - 1]);
        }
        jitter = diffSum / (pings.length - 1);
      }

      els.valPingMin.textContent = min.toFixed(1);
      els.valJitter.textContent = jitter.toFixed(1);

      if (gaugeEngine) gaugeEngine.setValue(avg, "ms", t("ping_title"));

      this.setUIStage("ping", 15 + Math.round(((i + 1) / this.pingCount) * 15));
      await new Promise((r) => setTimeout(r, 60));
    }

    els.cardPing.classList.remove("active");
    const finalAvg = pings.reduce((a, b) => a + b, 0) / pings.length;
    els.valPing.textContent = finalAvg.toFixed(1);
    let finalJitter = 0;
    if (pings.length > 1) {
      let diffSum = 0;
      for (let j = 1; j < pings.length; j++) diffSum += Math.abs(pings[j] - pings[j - 1]);
      finalJitter = diffSum / (pings.length - 1);
    }
    return { avg: finalAvg, jitter: finalJitter };
  }

  async runDownloadPhase() {
    this.setUIStage("download", 35);
    els.cardDownload.classList.add("active");
    if (gaugeEngine) gaugeEngine.setMode("download", "Mbps", t("downloading"));
    this.showStatus(`${t("downloading")} (${this.duration}s)...`);

    const signal = this.abortController.signal;
    const testDurationMs = this.duration * 1000;
    const startTime = performance.now();
    let lastSampleTime = startTime;
    let deltaSampleBytes = 0;
    let receivedBytes = 0;
    let peakMbps = 0;
    let smoothedMbps = 0;

    const chunkSizeToRequest = 64 * 1024 * 1024;
    const downloadURL = this.target === "direct"
      ? `/api/speedtest/download?bytes=${chunkSizeToRequest}`
      : `/api/nodes/${this.target}/download?bytes=${chunkSizeToRequest}`;

    while (performance.now() - startTime < testDurationMs) {
      if (signal.aborted) throw new Error("aborted");

      const resp = await fetch(downloadURL, {
        method: "GET",
        headers: { ...adminHeaders(), "Cache-Control": "no-cache" },
        signal,
      });

      if (!resp.ok) {
        throw new Error(`HTTP ${resp.status}`);
      }

      if (!resp.body) {
        throw new Error("ReadableStream not supported");
      }

      const reader = resp.body.getReader();

      while (true) {
        if (signal.aborted) {
          await reader.cancel();
          throw new Error("aborted");
        }

        const { done, value } = await reader.read();
        if (done) break;

        receivedBytes += value.byteLength;
        deltaSampleBytes += value.byteLength;

        const now = performance.now();
        const deltaSampleTime = (now - lastSampleTime) / 1000;

        if (deltaSampleTime >= 0.5) {
          const instMbps = (deltaSampleBytes * 8) / (deltaSampleTime * 1_000_000);
          lastSampleTime = now;
          deltaSampleBytes = 0;

          if (smoothedMbps === 0) {
            smoothedMbps = instMbps;
          } else {
            smoothedMbps = smoothedMbps * 0.65 + instMbps * 0.35;
          }

          if (smoothedMbps > peakMbps) {
            peakMbps = smoothedMbps;
            els.valDownloadPeak.textContent = `${peakMbps.toFixed(1)} Mbps`;
          }

          const totalElapsed = (now - startTime) / 1000;
          els.valDownloadBytes.textContent = formatBytes(receivedBytes);

          if (gaugeEngine) gaugeEngine.setValue(smoothedMbps, "Mbps", t("dl_title"));
          if (chartEngine) chartEngine.addPoint("download", totalElapsed, smoothedMbps);

          const progress = Math.min(65, 35 + Math.round((totalElapsed / this.duration) * 30));
          this.setUIStage("download", progress);
        }

        if (now - startTime >= testDurationMs) {
          await reader.cancel();
          break;
        }
      }
    }

    const totalDuration = (performance.now() - startTime) / 1000;
    const finalMbps = (receivedBytes * 8) / (Math.max(0.001, totalDuration) * 1_000_000);
    els.valDownload.textContent = finalMbps >= 100 ? finalMbps.toFixed(1) : finalMbps.toFixed(2);
    els.valDownloadBytes.textContent = formatBytes(receivedBytes);

    if (gaugeEngine) gaugeEngine.setValue(finalMbps, "Mbps", t("completed"));
    els.cardDownload.classList.remove("active");
    return { mbps: finalMbps, bytes: receivedBytes };
  }

  async runUploadPhase() {
    this.setUIStage("upload", 70);
    els.cardUpload.classList.add("active");
    if (gaugeEngine) gaugeEngine.setMode("upload", "Mbps", t("uploading"));
    this.showStatus(`${t("uploading")} (${this.duration}s)...`);

    const signal = this.abortController.signal;
    const testDurationMs = this.duration * 1000;
    const uploadURL = this.target === "direct"
      ? "/api/speedtest/upload"
      : `/api/nodes/${this.target}/upload`;

    const chunkSize = 512 * 1024;
    const chunkBuffer = new Uint8Array(chunkSize);
    for (let i = 0; i < chunkSize; i++) chunkBuffer[i] = (i % 256);

    const startTime = performance.now();
    let lastSampleTime = startTime;
    let deltaSampleBytes = 0;
    let totalUploadedBytes = 0;
    let peakMbps = 0;
    let smoothedMbps = 0;

    while (performance.now() - startTime < testDurationMs) {
      if (signal.aborted) throw new Error("aborted");

      const resp = await fetch(uploadURL, {
        method: "POST",
        headers: {
          ...adminHeaders(),
          "Content-Type": "application/octet-stream",
          "Cache-Control": "no-cache",
        },
        body: chunkBuffer,
        signal,
      });

      if (!resp.ok) {
        throw new Error(`HTTP ${resp.status}`);
      }

      totalUploadedBytes += chunkSize;
      deltaSampleBytes += chunkSize;

      const now = performance.now();
      const deltaSampleTime = (now - lastSampleTime) / 1000;

      if (deltaSampleTime >= 0.5) {
        const instMbps = (deltaSampleBytes * 8) / (deltaSampleTime * 1_000_000);
        lastSampleTime = now;
        deltaSampleBytes = 0;

        if (smoothedMbps === 0) {
          smoothedMbps = instMbps;
        } else {
          smoothedMbps = smoothedMbps * 0.65 + instMbps * 0.35;
        }

        if (smoothedMbps > peakMbps) {
          peakMbps = smoothedMbps;
          els.valUploadPeak.textContent = `${peakMbps.toFixed(1)} Mbps`;
        }

        const totalElapsed = (now - startTime) / 1000;
        els.valUploadBytes.textContent = formatBytes(totalUploadedBytes);

        if (gaugeEngine) gaugeEngine.setValue(smoothedMbps, "Mbps", t("ul_title"));
        if (chartEngine) chartEngine.addPoint("upload", totalElapsed, smoothedMbps);

        const progress = Math.min(95, 70 + Math.round((totalElapsed / this.duration) * 25));
        this.setUIStage("upload", progress);
      }
    }

    const totalDuration = (performance.now() - startTime) / 1000;
    const finalMbps = (totalUploadedBytes * 8) / (Math.max(0.001, totalDuration) * 1_000_000);
    els.valUpload.textContent = finalMbps >= 100 ? finalMbps.toFixed(1) : finalMbps.toFixed(2);
    els.valUploadBytes.textContent = formatBytes(totalUploadedBytes);

    if (gaugeEngine) gaugeEngine.setValue(finalMbps, "Mbps", t("completed"));
    els.cardUpload.classList.remove("active");
    return { mbps: finalMbps, bytes: totalUploadedBytes };
  }

  setUIStage(stage, percent) {
    const stages = ["ready", "ping", "download", "upload", "finished"];
    const stageIdx = stages.indexOf(stage);

    document.querySelectorAll(".pipe-node").forEach((el, idx) => {
      el.classList.remove("active", "completed");
      if (idx < stageIdx) el.classList.add("completed");
      else if (idx === stageIdx) el.classList.add("active");
    });

    for (let i = 1; i <= 4; i++) {
      const line = document.getElementById(`line-${i}`);
      if (line) {
        line.classList.remove("active", "completed");
        if (i < stageIdx) line.classList.add("completed");
        else if (i === stageIdx) line.classList.add("active");
      }
    }

    if (els.meterBar) {
      els.meterBar.style.width = `${percent}%`;
    }
  }

  resetMetrics() {
    els.valPing.textContent = "--";
    els.valJitter.textContent = "--";
    els.valPingMin.textContent = "--";
    els.valDownload.textContent = "--";
    els.valDownloadPeak.textContent = "--";
    els.valDownloadBytes.textContent = "--";
    els.valUpload.textContent = "--";
    els.valUploadPeak.textContent = "--";
    els.valUploadBytes.textContent = "--";
    this.hideError();
  }

  showStatus(msg) {
    els.statusBanner.classList.remove("hidden");
    els.statusText.textContent = msg;
  }

  showError(msg) {
    els.errorBanner.classList.remove("hidden");
    els.errorText.textContent = msg;
  }

  hideError() {
    els.errorBanner.classList.add("hidden");
  }

  finishTest() {
    state.isTesting = false;
  }
}

const engine = new SpeedtestEngine();

// ================= 节点选择与列表系统 =================

function selectNode(targetId) {
  state.selectedTarget = String(targetId);
  els.nodeSelect.value = state.selectedTarget;

  if (state.selectedTarget === "direct") {
    els.targetStatusDot.className = "status-beacon beacon-online";
    els.targetLatencyBadge.textContent = t("rtt_local");
  } else {
    const node = state.nodes.find((n) => String(n.id) === state.selectedTarget);
    if (node) {
      els.targetStatusDot.className = `status-beacon ${node.last_status === "online" ? "beacon-online" : "beacon-error"}`;
      els.targetLatencyBadge.textContent = node.last_latency_ms != null ? `RTT: ${node.last_latency_ms.toFixed(1)} ms` : t("rtt_unknown");
    }
  }
}

function renderNodeSelect() {
  const cur = els.nodeSelect.value;
  els.nodeSelect.innerHTML = `<option value="direct">${t("direct_node")}</option>`;

  state.nodes.forEach((node) => {
    if (!node.enabled) return;
    const opt = document.createElement("option");
    opt.value = String(node.id);
    const lat = node.last_latency_ms != null ? ` (${node.last_latency_ms.toFixed(1)}ms)` : "";
    opt.textContent = `${node.name}${lat}`;
    els.nodeSelect.appendChild(opt);
  });

  if (cur && Array.from(els.nodeSelect.options).some((o) => o.value === cur)) {
    els.nodeSelect.value = cur;
  }
}

function pickLowestLatencyNode() {
  const onlineNodes = state.nodes.filter(
    (n) => n.enabled && n.last_status === "online" && n.last_latency_ms != null
  );

  if (onlineNodes.length === 0) {
    showToast(t("rtt_unknown"), "info");
    selectNode("direct");
    return;
  }

  onlineNodes.sort((a, b) => a.last_latency_ms - b.last_latency_ms);
  const best = onlineNodes[0];
  selectNode(best.id);
  showToast(t("toast_opt_node", { name: best.name, lat: best.last_latency_ms.toFixed(1) }), "success");
}

// ================= 节点列表与数据渲染 =================

async function refreshNodes(silent = false) {
  if (!silent) {
    els.nodesLoading.classList.remove("hidden");
    els.nodesEmpty.classList.add("hidden");
  }

  try {
    const data = await apiFetch("/api/nodes", { headers: adminHeaders() });
    state.nodes = data.nodes || [];
    renderNodeSelect();
    renderNodesTable();
    updateSummaryStats();
  } catch (err) {
    if (!silent) {
      if (err.status === 401 || String(err.message).includes("令牌") || String(err.message).includes("token")) {
        els.nodesTableBody.innerHTML = `<tr><td colspan="8" style="text-align:center;color:var(--text-muted);padding:2rem;">
          ${state.lang === "zh" ? "当前处于访客测速模式（可直接执行中心测速）。如需管理节点矩阵，请在右上角配置 Admin Token。" : "Guest speedtest mode. To manage telemetry matrix, configure Admin Token."}
        </td></tr>`;
      } else {
        els.nodesTableBody.innerHTML = `<tr><td colspan="8" style="text-align:center;color:var(--neon-red);padding:2rem;">${escapeHtml(err.message)}</td></tr>`;
        showToast(`${err.message}`, "error");
      }
    }
    renderNodeSelect();
    updateSummaryStats();
  } finally {
    els.nodesLoading.classList.add("hidden");
  }
}

function updateSummaryStats() {
  const total = state.nodes.length;
  const online = state.nodes.filter((n) => n.last_status === "online").length;
  els.statTotalCount.textContent = total;
  els.statOnlineCount.textContent = online;
  els.nodesCountBadge.textContent = String(total);
}

function renderNodesTable() {
  const tbody = els.nodesTableBody;
  tbody.innerHTML = "";

  const q = state.searchQuery.toLowerCase().trim();
  const filter = state.activeFilter;

  const filtered = state.nodes.filter((node) => {
    if (filter === "online" && node.last_status !== "online") return false;
    if (filter === "enabled" && !node.enabled) return false;
    if (filter === "error" && node.last_status !== "error") return false;

    if (q) {
      const matchName = (node.name || "").toLowerCase().includes(q);
      const matchAddr = (node.address || "").toLowerCase().includes(q);
      const matchFp = (node.key_fingerprint || "").toLowerCase().includes(q);
      const matchPort = String(node.port).includes(q);
      if (!matchName && !matchAddr && !matchFp && !matchPort) return false;
    }
    return true;
  });

  if (filtered.length === 0) {
    els.nodesEmpty.classList.remove("hidden");
    els.emptyMessage.textContent = state.nodes.length === 0 ? t("nodes_empty") : t("nodes_empty");
    return;
  }

  els.nodesEmpty.classList.add("hidden");

  filtered.forEach((node) => {
    const tr = document.createElement("tr");
    tr.dataset.id = node.id;

    let statusBeacon = "beacon-error";
    let statusText = t("status_unknown");
    if (node.last_status === "online") {
      statusBeacon = "beacon-online";
      statusText = t("status_online");
    } else if (node.last_status === "error") {
      statusBeacon = "beacon-error";
      statusText = t("status_offline");
    }

    const latencyDisplay = node.last_latency_ms != null ? `${node.last_latency_ms.toFixed(1)} ms` : "--";

    tr.innerHTML = `
      <td><span style="font-family:var(--font-mono);color:var(--text-muted);">#${node.id}</span></td>
      <td><strong>${escapeHtml(node.name)}</strong></td>
      <td><code style="font-family:var(--font-mono);">${escapeHtml(node.protocol)}://${escapeHtml(node.address)}:${node.port}</code></td>
      <td><span style="font-family:var(--font-mono);font-size:0.75rem;color:var(--neon-cyan);">${escapeHtml(node.key_fingerprint)}</span></td>
      <td>
        <span style="display:inline-flex;align-items:center;gap:0.4rem;">
          <span class="status-beacon ${statusBeacon}"></span>
          ${statusText}
        </span>
      </td>
      <td><span>${node.enabled ? t("enabled") : t("disabled")}</span></td>
      <td><strong style="font-family:var(--font-mono);">${latencyDisplay}</strong></td>
      <td class="col-right">
        <button class="hud-action-link" data-action="speedtest" title="${t("act_test")}">${t("act_test")}</button>
        <button class="hud-action-link" data-action="health" title="${t("act_health")}">${t("act_health")}</button>
        <button class="hud-action-link" data-action="toggle">${node.enabled ? t("disabled") : t("enabled")}</button>
        <button class="hud-danger-link" data-action="delete">${t("act_del")}</button>
      </td>
    `;
    tbody.appendChild(tr);
  });
}

// ================= 节点操作与弹窗逻辑 =================

async function handleNodeAction(nodeId, action) {
  const node = state.nodes.find((n) => String(n.id) === String(nodeId));
  if (!node) return;

  if (action === "speedtest") {
    selectNode(nodeId);
    window.scrollTo({ top: 0, behavior: "smooth" });
    startSpeedtest();
  } else if (action === "health") {
    showToast(`${t("act_health")} [${node.name}]...`, "info");
    try {
      const res = await apiFetch(`/api/nodes/${nodeId}/health`, {
        method: "POST",
        headers: adminHeaders(),
      });
      if (res.status === "online") {
        showToast(`[${node.name}] RTT: ${res.latency_ms.toFixed(1)} ms`, "success");
      } else {
        showToast(`[${node.name}] ${res.error || "error"}`, "error");
      }
      refreshNodes(true);
    } catch (err) {
      showToast(`${err.message}`, "error");
    }
  } else if (action === "toggle") {
    const endpoint = node.enabled ? "disable" : "enable";
    try {
      await apiFetch(`/api/nodes/${nodeId}/${endpoint}`, {
        method: "POST",
        headers: adminHeaders(),
      });
      showToast(`[${node.name}] -> ${node.enabled ? t("disabled") : t("enabled")}`, "success");
      refreshNodes(true);
    } catch (err) {
      showToast(`${err.message}`, "error");
    }
  } else if (action === "delete") {
    if (!confirm(`${t("act_del")} [${node.name}]?`)) return;
    try {
      await apiFetch(`/api/nodes/${nodeId}`, {
        method: "DELETE",
        headers: adminHeaders(),
      });
      showToast(t("toast_node_deleted"), "success");
      if (state.selectedTarget === String(nodeId)) {
        selectNode("direct");
      }
      refreshNodes(true);
    } catch (err) {
      showToast(`${err.message}`, "error");
    }
  }
}

async function registerNewNode(e) {
  e.preventDefault();
  const name = els.regName.value.trim();
  const address = els.regAddress.value.trim();
  const port = parseInt(els.regPort.value, 10);
  const protocol = els.regProtocol.value;
  const metadataRaw = els.regMetadata.value.trim();

  let metadata = null;
  if (metadataRaw) {
    try {
      metadata = JSON.parse(metadataRaw);
    } catch {
      showToast("Metadata JSON invalid", "error");
      return;
    }
  }

  const payload = { name, address, port, protocol, metadata };

  try {
    const res = await apiFetch("/api/register", {
      method: "POST",
      headers: registrationHeaders(),
      body: payload,
    });

    els.formRegister.reset();
    switchTab("nodes");

    els.displayNewKey.textContent = res.node_key;
    els.copyKeyText.textContent = t("key_copy_btn");
    els.modalOneTimeKey.showModal();

    showToast(t("toast_node_registered"), "success");
    await refreshNodes(true);
    selectNode(res.id);
  } catch (err) {
    showToast(`${err.message}`, "error");
  }
}

// ================= 选项卡切换 =================

function switchTab(tabId) {
  els.tabBtns.forEach((btn) => {
    btn.classList.toggle("active", btn.dataset.tab === tabId);
  });
  els.tabPanes.forEach((pane) => {
    pane.classList.toggle("active", pane.id === `pane-${tabId}`);
  });

  if (els.tabActionsHistory) {
    els.tabActionsHistory.classList.toggle("hidden", tabId !== "history");
  }
  if (els.tabActionsNodes) {
    els.tabActionsNodes.classList.toggle("hidden", tabId !== "nodes");
  }
}

// ================= 事件监听绑定 =================

function startSpeedtest() {
  if (state.isTesting) return;
  state.isTesting = true;
  engine.run(state.selectedTarget);
}

function initEventListeners() {
  els.btnLangToggle.addEventListener("click", toggleLanguage);
  els.btnThemeToggle.addEventListener("click", toggleTheme);

  // 测速触发与取消
  els.btnStartTest.addEventListener("click", startSpeedtest);
  els.btnRetestCircle.addEventListener("click", startSpeedtest);
  els.btnCancelTest.addEventListener("click", () => engine.abort());
  els.btnDismissError.addEventListener("click", () => engine.hideError());

  // 节点选择与时长选择
  els.nodeSelect.addEventListener("change", (e) => {
    selectNode(e.target.value);
  });

  els.durationSelect.addEventListener("change", (e) => {
    state.testDuration = parseInt(e.target.value, 10) || 15;
    localStorage.setItem("ls_duration", String(state.testDuration));
    if (chartEngine) {
      chartEngine.maxTime = state.testDuration;
      chartEngine.reset();
    }
    showToast(t("toast_duration_set", { dur: state.testDuration }), "info");
  });

  els.btnPickLowest.addEventListener("click", pickLowestLatencyNode);

  // 选项卡切换
  els.tabBtns.forEach((btn) => {
    btn.addEventListener("click", () => switchTab(btn.dataset.tab));
  });

  // 历史记录交互
  els.btnClearHistory.addEventListener("click", clearAllHistory);
  els.btnExportHistory.addEventListener("click", exportHistoryCSV);
  els.historyTableBody.addEventListener("click", (e) => {
    const btn = e.target.closest("button[data-action]");
    if (!btn) return;
    const action = btn.dataset.action;
    if (action === "retest") {
      const target = btn.dataset.target;
      selectNode(target);
      window.scrollTo({ top: 0, behavior: "smooth" });
      startSpeedtest();
    } else if (action === "del-hist") {
      const id = btn.dataset.id;
      deleteHistoryItem(id);
    }
  });

  // 刷新与搜索
  els.btnRefresh.addEventListener("click", () => refreshNodes());
  els.filterSearch.addEventListener("input", (e) => {
    state.searchQuery = e.target.value;
    renderNodesTable();
  });

  // 筛选标签
  els.filterPills.forEach((btn) => {
    btn.addEventListener("click", () => {
      els.filterPills.forEach((b) => b.classList.remove("active"));
      btn.classList.add("active");
      state.activeFilter = btn.dataset.filter;
      renderNodesTable();
    });
  });

  // 节点表格事件代理
  els.nodesTableBody.addEventListener("click", (e) => {
    const btn = e.target.closest("button[data-action]");
    if (!btn) return;
    const row = btn.closest("tr");
    if (!row) return;
    const nodeId = row.dataset.id;
    const action = btn.dataset.action;
    handleNodeAction(nodeId, action);
  });

  // 注册表单提交
  els.formRegister.addEventListener("submit", registerNewNode);

  // 一次性 Key 弹窗
  els.btnCopyKey.addEventListener("click", async () => {
    const key = els.displayNewKey.textContent;
    try {
      await navigator.clipboard.writeText(key);
      els.copyKeyText.textContent = t("key_copied");
      showToast(t("toast_copied"), "success");
    } catch {
      showToast(t("toast_copy_fail"), "info");
    }
  });
  els.btnCloseKeyModal.addEventListener("click", () => {
    els.modalOneTimeKey.close();
    els.displayNewKey.textContent = "";
  });

  // 令牌弹窗
  els.btnOpenTokens.addEventListener("click", () => {
    els.inputAdminToken.value = state.adminToken;
    els.inputRegToken.value = state.registrationToken;
    els.modalTokens.showModal();
  });
  els.btnCloseTokens.addEventListener("click", () => els.modalTokens.close());
  els.btnClearTokens.addEventListener("click", () => {
    clearTokens();
    els.modalTokens.close();
  });
  els.formTokens.addEventListener("submit", (e) => {
    e.preventDefault();
    saveTokens(els.inputAdminToken.value, els.inputRegToken.value);
    els.modalTokens.close();
    refreshNodes();
  });

  // 密码显示/隐藏切换
  document.querySelectorAll(".btn-toggle-pw").forEach((btn) => {
    btn.addEventListener("click", () => {
      const targetInput = document.getElementById(btn.dataset.target);
      if (!targetInput) return;
      if (targetInput.type === "password") {
        targetInput.type = "text";
      } else {
        targetInput.type = "password";
      }
    });
  });

  // 响应式窗口缩放适配 HiDPI
  window.addEventListener("resize", () => {
    if (gaugeEngine) gaugeEngine.resize();
    if (chartEngine) chartEngine.resize();
  });
}

// ================= 初始化入口 =================
document.addEventListener("DOMContentLoaded", () => {
  // 1. 初始化多语言系统 (根据浏览器语言自动判定)
  initLanguage();

  // 2. 加载测速时长偏好 (默认 15s)
  const savedDuration = localStorage.getItem("ls_duration");
  if (savedDuration) {
    state.testDuration = parseInt(savedDuration, 10) || 15;
    if (els.durationSelect) {
      els.durationSelect.value = String(state.testDuration);
    }
  }

  // 3. 初始化主题与图表
  initTheme();
  gaugeEngine = new SpeedometerGauge(els.gaugeCanvas);
  chartEngine = new SpeedChart(els.chartCanvas);
  chartEngine.maxTime = state.testDuration;

  // 4. 加载数据并绑定事件
  loadTokens();
  loadHistory();
  initEventListeners();
  refreshNodes();
});
