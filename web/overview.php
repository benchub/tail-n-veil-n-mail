<?php
$days = 1;

if(isset($_GET["days"]))
{
	if(is_numeric($_GET["days"]))
	{
		if($_GET["days"] > 0)
		{
			$days = $_GET["days"];
		}
	}
}


// Connecting, selecting database
$dbconn = pg_connect("host=blahblah dbname=tnvnm user=www password=blahblah")
    or die('Could not connect: ' . pg_last_error());

// how many days ago did our current sprint start?
// '2013-12-28' is the day one sprint started, and they restart every 3 weeks
$query = "SELECT extract(days from current_date-(select date '2013-12-28' + interval '3 weeks' * ((extract(epoch from current_date)-extract(epoch from date '2013-12-28'))::int / 1728000)))";
$daysInSprint = pg_query($query) or die('Query failed: ' . pg_last_error());

echo "<h2>Showing log counts over the last $days days</h2>\n";

echo "show instead: <pre>";
echo "<a href='overview.php?days=.042'> 1 hour |</a>";
echo "<a href='overview.php?days=1'> 1 day |</a>";
echo "<a href='overview.php?days=7'> 1 week |</a>";
echo "<a href='overview.php?days=", pg_fetch_result($daysInSprint,0,0), "'> this sprint |</a>";
echo "<a href='overview.php?days=90'> 1 quarter</a>";
echo "</pre>";

// Performing SQL query
$query = "SELECT count(*),name,bucket_id from events,buckets where bucket_id=buckets.id and events.finished > now()-interval '1 day'*$days group by name,bucket_id order by name";
$namedThings = pg_query($query) or die('Query failed: ' . pg_last_error());
$query = "SELECT count(*) from events where bucket_id is null and events.finished > now()-interval '1 day'*$days";
$interestingThings = pg_query($query) or die('Query failed: ' . pg_last_error());

// Printing results in HTML
echo "<table>\n";
    echo "\t<tr>\n";
        echo "\t\t<td>Count</td>\n";
        echo "\t\t<td>Bucket</td>\n";
    echo "\t</tr>\n";
while ($row = pg_fetch_row($namedThings))
{
    echo "\t<tr>\n";
        echo "\t\t<td><a href='bucketDetails.php?days=$days&id=$row[2]'>$row[0]</a></td>\n";
	if(preg_match('/^CNVS-\d+$/',$row[1]))
	{
	        echo "\t\t<td><a href='https://instructure.atlassian.net/browse/$row[1]'>$row[1]</a></td>\n";
	}
	else
	{
	        echo "\t\t<td>" . htmlentities($row[1], ENT_QUOTES) . "</td>\n";
	}
    echo "\t</tr>\n";
}
while ($row = pg_fetch_row($interestingThings))
{
    echo "\t<tr>\n";
        echo "\t\t<td><a href='bucketDetails.php?days=$days'>$row[0]</a></td>\n";
        echo "\t\t<td><b>interesting things</b></td>\n";
    echo "\t</tr>\n";
}
echo "</table>\n";

// Free resultset
pg_free_result($namedThings);
pg_free_result($interestingThings);


// Closing connection
pg_close($dbconn);
?>
