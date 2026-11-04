import React, { useEffect, useState } from 'react';
import { PageContainer } from '@ant-design/pro-components';
import { Alert, Button, Card, Empty, Pagination, Popconfirm, Select, Space, Spin, Tag, Tooltip, Typography, message } from 'antd';
import { PlayCircleOutlined, ReloadOutlined } from '@ant-design/icons';
import { t, useI18n } from '@/i18n';
import { collectMediaSource, listMediaSchedules, listMediaSources, type MediaSource, type Schedule } from '@/services/media';
import { modeNames, StatusTag, timeText } from '../shared';
import ScheduleEditor from '../ScheduleEditor';

export default function Schedules() {
 useI18n();
 const [sources,setSources]=useState<MediaSource[]>([]),[plans,setPlans]=useState<Schedule[]>([]),[loading,setLoading]=useState(false),[editing,setEditing]=useState<Schedule>(),[filter,setFilter]=useState<number>(),[page,setPage]=useState(1),[starting,setStarting]=useState<string>();
 const load=async()=>{setLoading(true);try{const [a,b]=await Promise.all([listMediaSources(),listMediaSchedules()]);if(a.code!==0||b.code!==0)throw Error();setSources(a.data||[]);setPlans(b.data||[])}catch{message.error('加载采集计划失败')}finally{setLoading(false)}};
 useEffect(()=>{void load()},[]);
 // One-off runs share the queue with scheduled ones; a source runs one task at a time.
 const runNow=async(source:MediaSource,kind:string)=>{const key=`${source.id}:${kind}`;setStarting(key);try{const r=await collectMediaSource(source.id,kind);if(r.code!==0)throw Error(r.msg);message.success(<span>{t('cms.media.schedules.queued')} <a href="/#/media/runs">{t('cms.media.schedules.viewLog')}</a></span>)}catch(e){message.error(e instanceof Error&&e.message?e.message:t('cms.media.schedules.queueFailed'))}finally{setStarting(undefined)}};
 const blocked=(source:MediaSource,kind:string)=>!source.enabled?t('cms.media.schedules.sourceDisabled'):kind==='full'&&!source.fullEnabled?t('cms.media.schedules.fullNotAllowed'):'';
 return <PageContainer title={false}><div className="media-workspace">
 <Alert type="info" showIcon message={t('cms.media.schedules.guideTitle')} description={<ol style={{margin:0,paddingLeft:20}}><li>{t('cms.media.schedules.guideFull')}</li><li>{t('cms.media.schedules.guideIncremental')}</li><li>{t('cms.media.schedules.guideCalibrate')}</li></ol>} style={{marginBottom:24}}/>
 <div className="media-toolbar"><Select allowClear placeholder="全部站点" style={{width:240}} value={filter} onChange={value=>{setFilter(value);setPage(1)}} options={sources.map(s=>({value:s.id,label:s.name}))}/><Space><Typography.Text type="secondary">所有时间均为北京时间</Typography.Text><Button icon={<ReloadOutlined/>} loading={loading} onClick={load}>刷新</Button></Space></div>
 <Spin spinning={loading}><Space direction="vertical" size={20} style={{width:'100%'}}>
 {sources.filter(s=>!filter||s.id===filter).slice((page-1)*10,page*10).map(source=><Card className="media-panel" key={source.id} title={<Space><span className="media-source-dot">{source.name.slice(0,1)}</span>{source.name}</Space>} extra={<Tag color={source.enabled?'green':'default'}>{source.enabled?'站点已启用':'站点已停用'}</Tag>}>
 {!source.enabled&&<Alert message="站点停用期间，定时任务将被跳过。" type="warning" style={{marginBottom:16}}/>}
 {['incremental','full'].map(kind=>{const p=plans.find(s=>s.sourceId===source.id&&s.kind===kind);return p?<div key={kind} className="media-schedule-row">
 <div><Space><Typography.Text strong>{modeNames[kind]}</Typography.Text><Tag color={p.enabled?'blue':'default'}>{p.enabled?'已启用':'未启用'}</Tag></Space><div className="media-muted" style={{marginTop:8}}>{kind==='incremental'?'跟进新增与更新，适合高频执行':'遍历目录并校准，适合低频执行'}</div></div>
 <div><Typography.Text code>{p.cron}</Typography.Text><div className="media-muted" style={{marginTop:8}}>下次：{p.enabled?timeText(p.nextRunAt):'未安排'}</div></div>
 <div><StatusTag value={p.lastStatus}/><div className="media-muted" style={{marginTop:8}}>{timeText(p.lastRunAt)}</div>{p.lastError&&<Typography.Text type="danger" ellipsis={{tooltip:p.lastError}} style={{maxWidth:170}}>{p.lastError}</Typography.Text>}</div>
 <Space direction="vertical" size={8}><Tooltip title={blocked(source,kind)}><Popconfirm disabled={!!blocked(source,kind)} title={kind==='full'?t('cms.media.schedules.confirmFull'):t('cms.media.schedules.confirmIncremental')} description={<div style={{maxWidth:280}}>{kind==='full'?t('cms.media.schedules.confirmFullDetail'):t('cms.media.schedules.confirmIncrementalDetail')}</div>} onConfirm={()=>runNow(source,kind)}><Button type={kind==='full'?'primary':'default'} ghost={kind==='full'} block icon={<PlayCircleOutlined/>} disabled={!!blocked(source,kind)} loading={starting===`${source.id}:${kind}`}>{t('cms.media.schedules.runNow')}</Button></Popconfirm></Tooltip><Button block onClick={()=>setEditing(p)}>设置计划</Button></Space>
 </div>:<Empty key={kind} image={Empty.PRESENTED_IMAGE_SIMPLE} description={modeNames[kind]+'计划尚未初始化'}/>})}
 </Card>)}
 {!sources.length&&!loading&&<Empty description="请先添加资源站点"/>}{sources.filter(s=>!filter||s.id===filter).length>10&&<div className="media-pagination"><Pagination current={page} pageSize={10} total={sources.filter(s=>!filter||s.id===filter).length} showSizeChanger={false} onChange={setPage}/></div>}
 </Space></Spin>
 <ScheduleEditor schedule={editing} sourceName={sources.find(s=>s.id===editing?.sourceId)?.name} fullAllowed={sources.find(s=>s.id===editing?.sourceId)?.fullEnabled} onClose={()=>setEditing(undefined)} onSaved={()=>void load()}/>
 </div></PageContainer>;
}
