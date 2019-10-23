rootdir = "QUIC/";
frame = get_xy(rootdir + "frame.dat");
mean(frame(:,2))
std(frame(:,2))

probe = get_xy(rootdir + "probe.dat");

xack  = get_xy(rootdir + "xack.dat");

xack2  = get_xy(rootdir + "xack2.dat");

xack3  = get_xy(rootdir + "xack3.dat");

% dlmwrite(rootdir+"frame_latency.txt",frame)
% dlmwrite(rootdir+"server_avg_latency.txt",xack)

figure(1)
hold on
plot(frame(:,1),frame(:,2),'r','LineWidth',2)
% plot(probe(:,1),probe(:,2),'g','LineWidth',2)
% plot(xack(:,1),xack(:,2),'b','LineWidth',2)
plot(xack2(2:length(xack2),1),diff(xack2(:,2))*0.03-0.4,'g','LineWidth',2)
plot(xack3(2:length(xack3),1),diff(xack3(:,2))*0.03-0.15,'b','LineWidth',2)
legend('transmission latency','probe index increment','frame index increment')
hold off

fst = get_xy(rootdir + "frame_sent_time.dat");
fst_x = fst(:,1);fst_y = fst(:,2);
fst_y = fst_y-fst_y(1);
pst = get_xy(rootdir + "ping_sent_time.dat");
pst_x = pst(:,1);pst_y = pst(:,2);
pst_y = pst_y-pst_y(1);

figure(2)
hold on
scatter(fst_x,fst_y,140,'+');
scatter(pst_x,pst_y,100,'.');
legend('frame','probe')




