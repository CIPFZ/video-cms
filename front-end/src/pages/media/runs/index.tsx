import React, { useEffect, useState } from 'react';
import { PageContainer } from '@ant-design/pro-components';
import { Alert, Button, Card, Descriptions, Drawer, List, Popconfirm, Select, Space, Table, Tag, Typography, message } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import { cancelCollectionRun, listCollectionRuns, listMediaSources, type CollectionRun, type MediaSource } from '@/services/media';
import { t, useI18n } from '@/i18n';
import { modeNames, StatusTag, statusNames, timeText } from '../shared';
import RunItems from './RunItems';

export default function Runs() {
 useI18n();
 const [runs,setRuns]=useState<CollectionRun[]>([]),[total,setTotal]=useState(0),[summary,setSummary]=useState<Record<string,number>>({}),[page,setPage]=useState(1),[sources,setSources]=useState<MediaSource[]>([]),[loading,setLoading]=useState(false),[source,setSource]=useState<number>(),[mode,setMode]=useState<string>(),[status,setStatus]=useState<string>(),[selected,setSelected]=useState<number>(),[pollError,setPollError]=useState(false);
 // Filtering and paging happen on the server; health checks are kept in their own history.
 const load=async(silent=false)=>{if(!silent)setLoading(true);try{const r=await listCollectionRuns({page,pageSize:10,sourceId:source,mode,status});if(r.code!==0)throw Error();setRuns(r.data?.list||[]);setTotal(r.data?.total||0);setSummary(r.data?.summary||{});setPollError(false)}catch{setPollError(true)}finally{if(!silent)setLoading(false)}};
 useEffect(()=>{listMediaSources().then(r=>{if(r.code===0)setSources(r.data||[])}).catch(()=>message.error('站点名称加载失败'))},[]);
 useEffect(()=>{void load();const timer=window.setInterval(()=>void load(true),5000);return()=>clearInterval(timer)},[page,source,mode,status]);
 const active=runs.find(r=>r.id===selected);
 const cancel=async(id:number)=>{try{const r=await cancelCollectionRun(id);if(r.code!==0)throw Error(r.msg);message.success('已请求取消，任务将在当前步骤完成后停止');await load(true)}catch(e){message.error(e instanceof Error?e.message:'取消失败')}};
 const name=(id:number)=>sources.find(s=>s.id===id)?.name||'站点 #'+id;
 const count=(...keys:string[])=>keys.reduce((n,k)=>n+(summary[k]||0),0);
 return <PageContainer title={false}><div className="media-workspace">
 <div className="media-stat-grid">{[['运行与排队',count('running','queued')],['已完成',count('success')],['需要关注',count('failed','partial')],['全部任务',Object.values(summary).reduce((a,b)=>a+b,0)]].map(([label,value])=><div className="media-stat" key={label}><span className="media-muted">{label}</span><b>{value}</b></div>)}</div>
 {pollError&&<Alert type="warning" showIcon message="暂时无法刷新日志，当前显示上次成功加载的数据。" style={{marginBottom:16}}/>}
 <Card className="media-panel"><div className="media-toolbar"><Space wrap>
 <Select allowClear placeholder="全部站点" style={{width:180}} value={source} onChange={v=>{setSource(v);setPage(1)}} options={sources.map(s=>({value:s.id,label:s.name}))}/>
 <Select allowClear placeholder={t('cms.media.runs.allModes')} style={{width:160}} value={mode} onChange={v=>{setMode(v);setPage(1)}} options={Object.entries(modeNames).filter(([value])=>value!=='health').map(([value,label])=>({value,label}))}/>
 <Select allowClear placeholder="全部状态" style={{width:150}} value={status} onChange={v=>{setStatus(v);setPage(1)}} options={['queued','running','success','partial','failed','cancelled'].map(value=>({value,label:statusNames[value]}))}/>
 </Space><Space><Typography.Text type="secondary">共 {total} 条记录</Typography.Text><Tag color="processing">每 5 秒更新</Tag><Button icon={<ReloadOutlined/>} onClick={()=>load()}>刷新</Button></Space></div>
 <Table<CollectionRun> rowKey="id" loading={loading} dataSource={runs} pagination={{current:page,pageSize:10,total,showSizeChanger:false,onChange:setPage}} scroll={{x:1050}} columns={[
 {title:'任务',render:(_,r)=><div><Typography.Text strong>{name(r.sourceId)}</Typography.Text><div className="media-muted">#{r.id} · {modeNames[r.mode]||r.mode}</div></div>},
 {title:'状态',render:(_,r)=><StatusTag value={r.status}/>},
 {title:'采集进度',render:(_,r)=><div><Typography.Text>{r.fetched} 条 · {r.pages} 页</Typography.Text><div className="media-muted">{['queued','running'].includes(r.status)?'下一页：'+r.nextPage:'处理结束'}</div></div>},
 {title:'处理结果',render:(_,r)=><div><span style={{color:'#389e0d'}}>+{r.insertedCount} 新增</span> · {r.updatedCount} 更新<div className="media-muted">{r.skippedCount} 跳过 · <span style={{color:r.errorCount?'#cf1322':undefined}}>{r.errorCount} 错误</span></div></div>},
 {title:'任务时间',render:(_,r)=><div>{timeText(r.startedAt)}<div className="media-muted">{r.finishedAt?'结束 '+timeText(r.finishedAt):r.status==='queued'?'等待执行':'正在执行'}</div></div>},
 {title:'操作',width:150,render:(_,r)=><Space><Button type="link" onClick={()=>setSelected(r.id)}>详情</Button>{['running','queued'].includes(r.status)&&<Popconfirm title="取消这个任务？" description="已采集的数据会保留。" onConfirm={()=>cancel(r.id)}><Button type="link" danger>取消</Button></Popconfirm>}</Space>}
 ]}/></Card>
 <Drawer width={560} title={'任务 #'+selected} open={!!selected} onClose={()=>setSelected(undefined)}>
 {active&&<><Space style={{marginBottom:24}}><StatusTag value={active.status}/><Tag>{modeNames[active.mode]}</Tag></Space><Descriptions column={1} bordered size="small" items={[
 {key:'source',label:'站点',children:name(active.sourceId)},{key:'start',label:'开始时间',children:timeText(active.startedAt)},{key:'end',label:'结束时间',children:timeText(active.finishedAt)},
 {key:'progress',label:'采集进度',children:active.fetched+' 条 / '+active.pages+' 页'},{key:'next',label:'下一页断点',children:active.nextPage||'已结束'},
 {key:'new',label:'新增 / 更新',children:active.insertedCount+' / '+active.updatedCount},{key:'skip',label:'跳过 / 错误',children:active.skippedCount+' / '+active.errorCount}
 ]}/><Typography.Title level={5} style={{marginTop:28}}>执行说明</Typography.Title><Alert showIcon type={active.errorMessage?'warning':'info'} message={active.errorMessage||'暂无异常记录。任务进度按已处理页数与条目数更新。'}/>{active.pageFailures?.length ? <><Typography.Title level={5} style={{marginTop:28}}>失败分页</Typography.Title><List size="small" bordered dataSource={active.pageFailures} renderItem={failure => <List.Item><Space direction="vertical" size={2}><Typography.Text strong>第 {failure.page} 页 · 已重试 {failure.attempts} 次</Typography.Text><Typography.Text type="secondary">{failure.errorMessage}</Typography.Text></Space></List.Item>} /></> : null}<Typography.Title level={5} style={{marginTop:28}}>{t('cms.media.runItems.title')}</Typography.Title><RunItems run={active}/></>}
 </Drawer></div></PageContainer>;
}
